# 開発環境

Windows / macOS の Docker Desktop 上で、Frontend・Internal API・External API・MySQL をまとめて起動するための開発環境です。ホスト OS へ Node.js、Go、MySQL を入れる必要はありません。

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

初回はイメージの取得と MySQL の初期化に少し時間がかかります。Frontend の Vite、Go の Air、MySQL が起動したら準備完了です。

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
```

Docker コンテナ間では `localhost` ではなく、Compose のサービス名を使います。

```text
MySQL:
db:3306

External API:
http://external-api:8081
```

Frontend から Internal API を呼ぶときは、相対パスを使います。

```typescript
fetch("/api/example")
```

Vite が `/api/*` を `http://internal-api:8080` へ proxy します。CORS 設定は不要です。

## 動作確認

ブラウザで Frontend を開くと、次の接続結果が表示されます。

- Frontend → Internal API (`/api/example`)
- Internal API → MySQL (`db:3306`)
- Internal API → External API (`http://external-api:8081`)

MySQL の初期データは `migrations/001_init.sql` が、ボリュームが空の初回起動時に自動実行されます。

## 停止

```bash
docker compose down
```

コンテナは止まりますが、MySQL のデータは named volume に残ります。

## DB を含む完全初期化

```bash
docker compose down -v
docker compose up --build
```

`-v` を付けると named volume も削除されます。MySQL のデータは消え、次回起動時に `001_init.sql` が再度実行されて初期状態に戻ります。

## ログ確認

全サービスのログ:

```bash
docker compose logs -f
```

特定サービスのみ:

```bash
docker compose logs -f internal-api
```

## サービス再起動

例:

```bash
docker compose restart internal-api
```

## 開発中のソース変更

ソースは bind mount されています。イメージを毎回 build し直す必要はありません。

- Frontend: Vite HMR が反映します
- Internal API / External API: Air が再ビルドしてプロセスを再起動します

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
