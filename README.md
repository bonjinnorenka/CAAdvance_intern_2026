# 開発環境

Windows / macOS の Docker Desktop 上で、Frontend・Internal API・exampleAdsAPI・MySQL・Redis Queue・reportGenerateBatch をまとめて起動するための開発環境です。ホスト OS へ Node.js、Go、MySQL、Redis を入れる必要はありません。

利用期間は短期間を想定しています。`docker compose up --build` で再現できれば十分です。

## 必要なもの

- Git
- Docker Desktop

## 初回起動

```bash
git clone <repository>
cd <repository>
```

必要な `.env` がある場合のみ作成してください。今回のサンプルは未設定でも起動します。

macOS / Linux / Git Bash:

```bash
cp .env.example .env
```

Windows PowerShell:

```powershell
Copy-Item .env.example .env
```

起動:

```bash
docker compose up --build
```

バックグラウンドで起動する場合:

```bash
docker compose up --build -d
```

初回はイメージの取得と MySQL の初期化に少し時間がかかります。Frontend、API、MySQL、Redis Queue、reportGenerateBatch が起動したら準備完了です。

Batch は常駐しません。必要なときに手動実行します。

## アクセス先

ホスト OS のブラウザやクライアントからは次のアドレスを使います。

```text
Frontend:
http://localhost:5173

Internal API:
http://localhost:8080

exampleAdsAPI:
http://localhost:8081

MySQL:
localhost:3306

Redis Queue:
localhost:6379
```

Docker コンテナ間では `localhost` ではなく、Compose のサービス名を使います。

```text
MySQL:
db:3306

Redis Queue:
queue:6379

exampleAdsAPI:
http://example-ads-api:8081
```

Frontend から Internal API を呼ぶときは、相対パスを使います。

```typescript
fetch("/api/example")
fetch("/api/jobs", { method: "POST" })
```

Vite が `/api/*` を `http://internal-api:8080` へ proxy します。

## exampleAdsAPI（架空媒体の仮API）

ExampleAds のデモ用モックです。認証キーはベタ書きです。

```http
Authorization: Bearer example-ads-demo-key
```

```bash
curl -H "Authorization: Bearer example-ads-demo-key" \
  "http://localhost:8081/v1/accounts"

curl -H "Authorization: Bearer example-ads-demo-key" \
  "http://localhost:8081/v1/reports?account_id=acc_00101&date_from=2026-07-01&date_to=2026-07-07"
```

- 広告アカウントは 10 件固定（`acc_00101` … `acc_00110`）
- レポート数値は `account_id` と日付から決まる固定 seed で毎回同じ値になります
- 前日分は毎日 AM 2:00（JST）に確定します。未確定日と未来日は `rows` に出ません
- レート制限はプロセス全体で 60 リクエスト / 分です（`/health` は対象外）

## サービスの役割

```text
Frontend → Internal API → MySQL / Redis Queue
reportGenerateBatch ← Redis Queue (report_generate) → MySQL / data/exports
Batch    → exampleAdsAPI / MySQL
```

exampleAdsAPI へアクセスするのはデータ取得 Batch だけです。Internal API、reportGenerateBatch、Frontend からは呼びません。

レポート作成の受付（権限チェック・Queue 投入・履歴 API）は Internal API の後続実装です。reportGenerateBatch は `report_generate` キューからジョブを取り出し、CSV を生成します。

## データ取得バッチ

ExampleAds から広告アカウントとレポートを取得し、MySQL に保存します。同一 `(ad_account_id, date)` は UPSERT されるため、再取得しても重複行は作られません。

レポートは JST AM 2:00 に前日分が確定します。本番では AM 3:00 以降のスケジュール実行を推奨します。

```mermaid
sequenceDiagram
    participant Scheduler as Scheduler_AM3JST
    participant Batch as DataFetchBatch
    participant API as ExampleAds
    participant DB as MySQL

    Scheduler->>Batch: 起動（デフォルト: 前日分 / --full-range: 全域）
    Batch->>DB: migrations 自動適用
    Batch->>API: GET /v1/accounts（ページング）
    alt 429 rate_limited
        Batch->>Batch: 指数バックオフでリトライ
    end
    Batch->>DB: ad_accounts UPSERT + 媒体に無い ID は is_deleted=true
    loop 各アクティブアカウント
        Batch->>API: GET /v1/reports?account_id&date_from&date_to
        alt 404 not_found
            Batch->>Batch: スキップして次アカウントへ
        else 429
            Batch->>Batch: リトライ
        end
        Batch->>DB: ad_data UPSERT（ad_account_id, date）
    end
    Batch-->>Scheduler: 終了
```

### Seed ユーザー

| 種別 | name | role | 紐づけ ad_account |
|------|------|------|-------------------|
| 管理者 | 管理者 | admin | acc_00101 〜 acc_00105 |
| 一般ユーザー | 一般ユーザー | user | acc_00106 〜 acc_00108 |

## レポート生成バッチ（非同期）

`reportGenerateBatch` は常駐し、Redis キュー `report_generate` を BLPOP します。CSV は `data/exports/report-{id}.csv` に保存し、`report.status` を更新します。

依頼の受付（画面 / Internal API）は今回の対象外です。後続 API が `report` を `status=queued` で記録し、次の JSON をキューへ投入する想定です。

```json
{ "id": "<任意>", "type": "generate_report", "payload": { "report_id": 1 } }
```

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant API as InternalAPI
    participant Queue as ReportGenerateQueue
    participant DB
    participant Batch as ReportGenerateBatch
    participant Storage as StorageCSV

    Note over User,API: 受付（画面・API）は後続実装
    User->>Frontend: アカウント/期間/マージンを入力して作成依頼
    Frontend->>API: POST /api/reports
    API->>DB: report を status=queued で記録
    API->>Queue: ジョブ投入
    API-->>Frontend: 202 job_id（ユーザーは待たない）
    Batch->>Queue: ジョブ取り出し
    Batch->>DB: status=processing
    Batch->>DB: 対象アカウントの ad_data を日別取得
    Batch->>Storage: マージン適用して CSV 保存
    alt 成功（データ無しはヘッダのみ）
        Batch->>DB: status=completed file_path
    else 失敗
        Batch->>DB: status=failed reason
    end
    Note over User,Frontend: 履歴・ダウンロードも後続の画面/API
```

顧客請求額 = 媒体費用 ÷ (1 − マージン料率 ÷ 100)。金額は税抜 JPY で、生成時に四捨五入して整数円にします。DB の `ad_data` には媒体費用のみ保存します。

手動でキューへ投入する例:

```bash
docker compose exec queue redis-cli RPUSH report_generate '{"id":"job-1","type":"generate_report","payload":{"report_id":1}}'
```

## Batch を手動実行

Batch は本番では指定時刻に実行される想定ですが、この開発環境では scheduler を使いません。

開発者が次のコマンドを実行した時点を「その時刻が来た」とみなします。

```bash
docker compose run --rm batch
```

前日分（デフォルト）以外の取得:

```bash
docker compose run --rm batch -- --full-range
docker compose run --rm batch -- --date=2026-09-01
```

処理が終わるとコンテナは終了します。

Batch は exampleAdsAPI からデータを取得し、MySQL へ保存します。

DB 確認例:

```bash
docker compose exec db mysql -uapp -papp app -e \
  "SELECT COUNT(*) FROM ad_accounts; SELECT * FROM ad_data LIMIT 5;"
docker compose exec db mysql -uapp -papp app -e \
  "SELECT u.name, u.role, p.ad_account_id FROM users u JOIN user_ad_account_permissions p ON u.id = p.user_id ORDER BY u.id, p.ad_account_id;"
```

## 停止

```bash
docker compose down
```

コンテナは止まりますが、MySQL のデータは named volume に残ります。Redis Queue のジョブはこの開発環境では永続化しません。

## ログ確認

全サービスのログ:

```bash
docker compose logs -f
```

reportGenerateBatch のみ:

```bash
docker compose logs -f report-generate-batch
```

## サービス再起動

例:

```bash
docker compose restart report-generate-batch
```

## DB を含む完全初期化

```bash
docker compose down -v
docker compose up --build
```

`-v` を付けると named volume も削除されます。MySQL のデータは消え、次回起動時に migrations が再度実行されて初期状態に戻ります。

Batch / Internal API / reportGenerateBatch 起動時にも `migrations/` 配下の未適用 SQL が自動適用されます（`schema_migrations` で管理）。

## Queue の中身を消したい場合

通常は不要です。残ったジョブを捨てたいときだけ実行してください。

```bash
docker compose exec queue redis-cli FLUSHALL
```

## 開発中のソース変更

ソースは bind mount されています。イメージを毎回 build し直す必要はありません。

- Frontend: Vite HMR が反映します
- Internal API / exampleAdsAPI / reportGenerateBatch: Air が再ビルドしてプロセスを再起動します
- Batch: 実行のたびに最新ソースを使います

Windows / macOS の Docker Desktop でも検知できるよう、ファイル監視は polling を使っています。

## ディレクトリ構成

```text
.
├── compose.yaml
├── README.md
├── .gitignore
├── .env.example
├── frontend/
├── internal-api/
├── exampleAdsAPI/
├── worker/                  # reportGenerateBatch のソース
├── batch/
├── data/exports/            # 生成 CSV（git 管理しない）
└── migrations/
    ├── 001_init.sql
    ├── 002_ads_schema.sql
    ├── 003_seed_users.sql
    └── 004_report_columns.sql
```

## DB 接続情報（ローカル開発専用）

```text
database: app
user: app
password: app
root password: root
```

この認証情報はローカル開発専用です。exampleAdsAPI のデモキーはソースにベタ書きしています。Batch 用のキーを使う場合は `.env` に置き、Git 管理しないでください。
