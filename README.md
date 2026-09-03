# 開発環境

Windows / macOS の Docker Desktop 上で、Frontend・Internal API・External API・MySQL・Redis Queue・Worker をまとめて起動するための開発環境です。ホスト OS へ Node.js、Go、MySQL、Redis を入れる必要はありません。

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

初回はイメージの取得と MySQL の初期化に少し時間がかかります。Frontend、API、MySQL、Redis Queue、Worker が起動したら準備完了です。

Batch は常駐しません。必要なときに手動実行します。

## アクセス先

ホスト OS のブラウザやクライアントからは次のアドレスを使います。

```text
Frontend:
http://localhost:5173

Internal API:
http://localhost:8080

External API:
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

External API:
http://external-api:8081
```

Frontend から Internal API を呼ぶときは、相対パスを使います。

```typescript
fetch("/api/example")
fetch("/api/jobs", { method: "POST" })
```

Vite が `/api/*` を `http://internal-api:8080` へ proxy します。

## サービスの役割

```text
Frontend → Internal API → MySQL / Redis Queue
Worker   ← Redis Queue  → MySQL
Batch    → External API / Redis Queue / MySQL
```

External API へアクセスするのは Batch だけです。Internal API、Worker、Frontend からは呼びません。

## Batch を手動実行

Batch は本番では指定時刻に実行される想定ですが、この開発環境では scheduler を使いません。

開発者が次のコマンドを実行した時点を「その時刻が来た」とみなします。

```bash
docker compose run --rm batch
```

処理が終わるとコンテナは終了します。

Batch は External API からデータを取得し、MySQL へ保存したあと Queue へジョブを投入します。重い処理は Worker が Queue から受け取って実行します。

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

Worker のみ:

```bash
docker compose logs -f worker
```

## サービス再起動

例:

```bash
docker compose restart worker
```

## DB を含む完全初期化

```bash
docker compose down -v
docker compose up --build
```

`-v` を付けると named volume も削除されます。MySQL のデータは消え、次回起動時に `001_init.sql` が再度実行されて初期状態に戻ります。

## Queue の中身を消したい場合

通常は不要です。残ったジョブを捨てたいときだけ実行してください。

```bash
docker compose exec queue redis-cli FLUSHALL
```

## 開発中のソース変更

ソースは bind mount されています。イメージを毎回 build し直す必要はありません。

- Frontend: Vite HMR が反映します
- Internal API / External API / Worker: Air が再ビルドしてプロセスを再起動します
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
├── external-api/
├── worker/
├── batch/
└── migrations/
    └── 001_init.sql
```

## DB 接続情報（ローカル開発専用）

```text
database: app
user: app
password: app
root password: root
```

この認証情報はローカル開発専用です。External API キーなどの Secret は `.env` に置き、Git 管理しないでください。
