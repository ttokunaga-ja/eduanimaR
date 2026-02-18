# Docker 環境 & マイクロサービス連携ガイド

## ⚡ 原則：**すべての起動は Docker Compose 経由**

このプロジェクトでは、開発・テスト・本番いずれの環境においても  
**すべてのサービス起動を `docker compose`（root Makefile）経由で管理する**。

- ローカルに Go / Python / Node をインストールしなくても開発できる
- 環境差異（OS・ランタイムバージョン）を排除する
- `make dev` 1 コマンドで全サービスが起動する

---

## サービス構成とポート一覧

| サービス       | 役割                     | ホスト公開ポート | Docker 内部ホスト   |
|--------------|--------------------------|----------------|-------------------|
| `frontend`   | Next.js 開発サーバー       | `3000`         | `frontend:3000`   |
| `professor`  | Go REST API バックエンド   | `8080`         | `professor:8080`  |
| `librarian`  | Python gRPC 推論サービス   | `50051`        | `librarian:50051` |
| `postgres`   | PostgreSQL + pgvector    | `5432`         | `postgres:5432`   |
| `minio`      | S3互換オブジェクトストレージ | `9000` / `9001`| `minio:9000`      |
| `kafka`      | メッセージブローカー        | `9094`         | `kafka:9092`      |

> 使用イメージ: `apache/kafka:3.7.0`（KRaftモード — ZooKeeper不要）。旧レガシーイメージ（3.7 系）は移動/非推奨のため、本リポジトリでは Apache 公式イメージに統一しています。

> **⚠️ 開発環境は単一ブローカー構成 — Kafka レプリケーション設定について**
>
> Kafka のデフォルト設定では `offsets.topic.replication.factor=3` になっており、  
> 単一ブローカー（開発 docker-compose）では `__consumer_offsets` の自動作成に失敗します。  
> この状態で Consumer を起動すると `Group Coordinator Not Available` エラーが繰り返し発生します。
>
> そのため `docker-compose.yml` の `kafka` サービスに以下のオーバーライドを設定しています（変更しないこと）:
> ```
> KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1
> KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1
> KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1
> ```
>
> **⚠️ 本番環境ではこの設定を使わないこと。** 本番 Kafka（またはフェーズ2以降の Cloud Pub/Sub）では  
> ブローカー数に合わせて `replication.factor` を適切に設定してください。
>
> Kafka が正常に起動しているか確認する場合は `make smoke-kafka` を実行してください。
>
> CI ガード: リポジトリには `infra-smoke` GitHub Actions ワークフロー（`.github/workflows/infra-smoke.yml`）を追加しており、`docker-compose` 起動後に `GET /healthz` と `GET /api/v1/subjects` を検証します。不要な設定の巻き戻しを防ぎます。

> **重要**: Docker ネットワーク内のサービス間通信はサービス名で解決する（例: `http://professor:8080`）。  
> ブラウザからアクセスする URL は `localhost` ベース（例: `http://localhost:8080`）。

---

## 起動方法

### 初回セットアップ

```bash
# リポジトリルートで実行
cp .env.example .env
vim .env   # GEMINI_API_KEY を設定（必須）
```

### 開発環境（推奨）

```bash
# 全サービス起動（フォアグラウンド・ログ表示）
make dev

# 全サービス起動（バックグラウンド）
make dev-d

# インフラのみ起動（DB・MinIO・Kafka）
make infra
```

### よく使うコマンド

```bash
make logs              # 全ログ追尾
make logs-professor    # Professor のログのみ
make ps                # コンテナ状態確認
make down              # 全コンテナ停止
make restart           # 全コンテナ再起動
make clean             # コンテナ + ボリューム削除（DB データも消える）
```

### DB マイグレーション

```bash
# postgres コンテナが起動している状態で実行
make migrate           # 未適用マイグレーションを適用
make migrate-dry       # 適用予定のマイグレーションを確認（適用しない）
```

### テスト実行

テストはコンテナ外（ローカル）で実行する（外部依存不要）:

```bash
make test-all          # 全テスト
make test-professor    # Go ユニットテスト
make test-librarian    # Python ユニットテスト
make test-frontend     # Vitest ユニットテスト
make test-e2e          # Playwright E2E（dev 起動中に実行）
```

### プロダクションビルド

```bash
make prod              # production ステージのイメージでビルド + 起動
make build-prod        # ビルドのみ（起動しない）
```

---

## 環境変数

`.env.example` をコピーして `.env` を作成する:

```bash
cp .env.example .env
```

| 変数名                        | 説明                              | デフォルト値                    |
|-----------------------------|----------------------------------|-------------------------------|
| `GEMINI_API_KEY`            | Google Gemini API キー（**必須**）  | —                             |
| `POSTGRES_USER`             | PostgreSQL ユーザー名              | `eduanima`                    |
| `POSTGRES_PASSWORD`         | PostgreSQL パスワード              | `eduanima_password`           |
| `POSTGRES_DB`               | データベース名                     | `eduanima_professor`          |
| `MINIO_ROOT_USER`           | MinIO アクセスキー                 | `minioadmin`                  |
| `MINIO_ROOT_PASSWORD`       | MinIO シークレットキー              | `minioadmin`                  |
| `MINIO_BUCKET`              | バケット名                         | `eduanima-materials`          |
| `KAFKA_TOPIC_INGEST`        | 取り込みジョブトピック名            | `eduanima.ingest.jobs`        |
| `PORT`                      | Professor HTTP ポート             | `8080`                        |
| `GRPC_PORT`                 | Librarian gRPC ポート             | `50051`                       |
| `PROFESSOR_MODEL_FAST`      | 高速モデル名                       | `gemini-2.0-flash`            |
| `PROFESSOR_MODEL_ACCURATE`  | 高精度モデル名                     | `gemini-2.5-pro`              |
| `LOG_LEVEL`                 | ログレベル                         | `info`（debug も可）           |
| `NEXT_PUBLIC_API_BASE_URL`  | ブラウザからの API ベース URL       | `http://localhost:8080/api`   |

> `.env` は gitignore 済み。`.env.example` のみリポジトリに含める。

---

## Docker Compose ファイル構成

```
docker-compose.yml        # 開発環境（デフォルト・ホットリロード）
docker-compose.prod.yml   # プロダクション上書き設定（make prod で使用）
```

| ファイル                    | 用途                                           |
|---------------------------|------------------------------------------------|
| `docker-compose.yml`      | 全サービス定義（dev ステージ・ボリュームマウント） |
| `docker-compose.prod.yml` | production ステージ上書き（ボリューム無効化）    |

> ⚠️ PostgreSQL 18（`pgvector/pgvector:pg18` 等）を使用する場合 **必ずデータボリュームを `/var/lib/postgresql` にマウント**してください。
>
> 理由: PG18 系イメージはメジャーバージョン別ディレクトリを内部で管理するため、既存の `/var/lib/postgresql/data` を直接マウントするとデータレイアウト不整合で起動に失敗します（`pg_upgrade` 要件）。開発環境では `/var/lib/postgresql` マウントを標準とし、本番移行時はバックアップ→pg_upgrade 等の手順を必ず実施してください。
>
> 参照: Docker-Postgres のマウント仕様（PG18+）とアップグレード注意点。

---

## URL / ポート早見表

| アクセス先                | URL                              |
|-------------------------|----------------------------------|
| Frontend（ブラウザ）      | http://localhost:3000            |
| Professor API           | http://localhost:8080/api/v1/... |
| MinIO Console           | http://localhost:9001            |
| PostgreSQL              | localhost:5432                   |
| Kafka（外部）            | localhost:9094                   |
| Librarian gRPC          | grpc://localhost:50051           |

---

## CORS / プロキシ方針

- **フロントエンドは Next.js の Route Handler（`src/app/proxy.ts`）を通じて Professor に中継**  
  → ブラウザから直接バックエンドを叩かないため CORS 設定不要
- Cookie 認証：`SameSite=Lax`（ローカル）/ `SameSite=Strict` + `Secure`（本番）
- ハードコード禁止：ホスト名・ポート番号はすべて環境変数経由で注入

---

## 禁止事項（AI 向け）

- Docker を使わずに直接 `go run` / `python` / `npm run dev` を開発手順として記述しない
- `docker-compose.yml` に記載されていないポート番号・ホスト名を推測で使わない
- API 仕様を推測でエンドポイントを手書きしない（OpenAPI → Orval 生成を優先）
- `.env` にシークレットをコミットしない
