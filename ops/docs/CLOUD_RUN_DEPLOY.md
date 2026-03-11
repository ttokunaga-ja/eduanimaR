# Cloud Run デプロイアーキテクチャ

> 運用導線（Phase 1）: `ops/README.md` / `ops/cloudrun/README.md`

> **ステータス**: Phase 2 設計ドキュメント（Phase 1 はローカル Docker Compose 完結）  
> **対象リポジトリ**: eduanimaR（モノレポ）  
> **クラウド**: Google Cloud Platform

---

## 目次

1. [全体アーキテクチャ](#全体アーキテクチャ)
2. [サービス構成](#サービス構成)
3. [ネットワーク設計](#ネットワーク設計)
4. [マネージドサービス接続](#マネージドサービス接続)
5. [シークレット管理](#シークレット管理)
6. [CI/CD パイプライン](#cicd-パイプライン)
7. [デプロイ手順](#デプロイ手順)
8. [環境変数リファレンス](#環境変数リファレンス)
9. [コスト最適化](#コスト最適化)

---

## 全体アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Google Cloud Platform                         │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    VPC (eduanima-vpc)                         │   │
│  │                                                               │   │
│  │  ┌────────────────┐      ┌─────────────────────────────────┐ │   │
│  │  │  Cloud Load    │      │       Cloud Run Services         │ │   │
│  │  │  Balancing     │      │                                  │ │   │
│  │  │  + Cloud CDN   │─────▶│  ┌──────────────────────────┐   │ │   │
│  │  └────────────────┘      │  │  eduanima-frontend        │   │ │   │
│  │           ▲              │  │  (Next.js 15 standalone)  │   │ │   │
│  │           │ HTTPS        │  │  PUBLIC                   │   │ │   │
│  │  ┌────────┴───────┐      │  └──────────┬───────────────┘   │ │   │
│  │  │  Cloud Armor   │      │             │ internal           │ │   │
│  │  │  (WAF/DDoS)    │      │  ┌──────────▼───────────────┐   │ │   │
│  │  └────────────────┘      │  │  eduanima-professor       │   │ │   │
│  │                          │  │  (Go 1.25 + Echo)         │   │ │   │
│  │  ┌────────────────┐      │  │  INTERNAL (LB経由のみ)    │   │ │   │
│  │  │  Cloud SQL     │◀─────│  └──────────┬───────────────┘   │ │   │
│  │  │  PostgreSQL 18 │      │             │ Cloud Run internal  │ │   │
│  │  │  + pgvector    │      │  ┌──────────▼───────────────┐   │ │   │
│  │  └────────────────┘      │  │  eduanima-librarian       │   │ │   │
│  │                          │  │  (Python 3.12 + LangGraph)│   │ │   │
│  │  ┌────────────────┐      │  │  INTERNAL ONLY            │   │ │   │
│  │  │  Cloud Storage │◀─────│  └──────────────────────────┘   │ │   │
│  │  │  (教材ファイル) │      └─────────────────────────────────┘ │   │
│  │  └────────────────┘                                           │   │
│  │                                                               │   │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐  │   │
│  │  │  Pub/Sub       │  │  Secret Manager│  │  Artifact Reg. │  │   │
│  │  │  (Kafka代替)   │  │  (API keys等)  │  │  (Docker imgs) │  │   │
│  │  └────────────────┘  └────────────────┘  └────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## サービス構成

### 1. eduanima-frontend（Next.js 15）

| 項目 | 設定値 |
|------|--------|
| イメージ | `asia-northeast1-docker.pkg.dev/[PROJECT]/eduanima/frontend:$TAG` |
| アクセス | **Public** (HTTPS, Cloud LB 経由) |
| リージョン | `asia-northeast1`（東京）|
| 最小インスタンス | 1（コールドスタート防止）|
| 最大インスタンス | 10 |
| CPU | 1 vCPU |
| メモリ | 512Mi |
| 同時接続数 | 80 |
| ポート | 3000 |

### 2. eduanima-professor（Go 1.25）

| 項目 | 設定値 |
|------|--------|
| イメージ | `asia-northeast1-docker.pkg.dev/[PROJECT]/eduanima/professor:$TAG` |
| アクセス | **Internal + Cloud LB** (frontend からのみ) |
| リージョン | `asia-northeast1` |
| 最小インスタンス | 1 |
| 最大インスタンス | 20 |
| 同時接続数 | 10（1 QAリクエスト ≈ 9 Gemini API calls）|
| CPU | 1 vCPU |
| メモリ | 512Mi |
| ポート | 8080 |

### 3. eduanima-librarian（Python 3.12）

| 項目 | 設定値 |
|------|--------|
| イメージ | `asia-northeast1-docker.pkg.dev/[PROJECT]/eduanima/librarian:$TAG` |
| アクセス | **Internal Only** (professor からのみ) |
| リージョン | `asia-northeast1` |
| 最小インスタンス | 1 |
| 最大インスタンス | 5 |
| CPU | 1 vCPU |
| メモリ | 1Gi |
| ポート | 50051 (gRPC) |

---

## ネットワーク設計

### アクセス制御マトリクス

| 送信元 | 送信先 | プロトコル | 許可 |
|--------|--------|-----------|------|
| ブラウザ | frontend | HTTPS (443) | ✅ Public |
| frontend | professor | HTTPS (443) | ✅ Internal LB |
| professor | librarian | gRPC (50051) | ✅ Internal |
| ブラウザ | professor | HTTPS (443) | ❌ 拒否 |
| ブラウザ | librarian | - | ❌ 拒否 |
| professor | Cloud SQL | TCP (5432) | ✅ Cloud SQL Proxy |
| professor | Cloud Storage | HTTPS | ✅ Workload Identity |
| professor | Pub/Sub | HTTPS | ✅ Workload Identity |

---

## マネージドサービス接続

### Phase 1 → Phase 2 移行マッピング

| ローカル（Phase 1） | Cloud（Phase 2） | 変更箇所 |
|---------------------|-----------------|---------|
| PostgreSQL + pgvector | Cloud SQL for PostgreSQL 18 + pgvector | `DATABASE_URL` のみ変更 |
| MinIO | Cloud Storage + HMAC 認証 | `OBJECT_STORAGE_BACKEND=gcs` |
| Kafka (KRaft) | Cloud Pub/Sub（Phase 2 移行） | Kafka クライアントを Pub/Sub SDK に置換 |

### Cloud SQL 接続

```yaml
# Cloud SQL Auth Proxy（Cloud Run で自動設定）
# cloud-sql-proxy 不要：--add-cloudsql-instances フラグで自動 Proxy
DATABASE_URL: postgres://user:pass@/eduanima_professor?host=/cloudsql/PROJECT:REGION:INSTANCE
```

### Cloud Storage（GCS）接続

```yaml
# Workload Identity で認証（サービスアカウントキー不要）
OBJECT_STORAGE_BACKEND: gcs
GCS_BUCKET: eduanima-materials-prod
# MINIO_* 変数は不要（GCS SDK が自動認証）
```

---

## シークレット管理

### Secret Manager に保存するシークレット一覧

```bash
# シークレット作成コマンド例
echo -n "YOUR_GEMINI_API_KEY" | \
  gcloud secrets create GEMINI_API_KEY \
    --data-file=- \
    --replication-policy=regional \
    --locations=asia-northeast1

# シークレット一覧
gcloud secrets create GEMINI_API_KEY         # Gemini API Key
gcloud secrets create professor-database-url  # Cloud SQL 接続文字列
```

### サービスアカウント設計

```
eduanima-frontend-sa:
  roles:
    - run.invoker (professor Cloud Run を呼び出す)
    - secretmanager.secretAccessor

eduanima-professor-sa:
  roles:
    - run.invoker (librarian Cloud Run を呼び出す)
    - cloudsql.client
    - storage.objectAdmin
    - pubsub.publisher
    - pubsub.subscriber
    - secretmanager.secretAccessor

eduanima-librarian-sa:
  roles:
    - secretmanager.secretAccessor
```

---

## CI/CD パイプライン

CI/CD は **Cloud Build** で管理します。  
設定ファイル: `ops/cloudrun/cloudbuild.yaml`

```
git push origin main
        │
        ▼
[Cloud Build トリガー起動]
        │
        ├── [1] build-librarian → push-librarian
        ├── [2] build-professor → push-professor      （並列）
        ├── [3] build-frontend  → push-frontend       （並列）
        │
        ├── [4] deploy-librarian  （librarian push 後）
        ├── [5] deploy-professor  （professor push + librarian deploy 後）
        └── [6] deploy-frontend   （frontend push + professor deploy 後）
```

手動デプロイ:

```bash
# 全サービス
make deploy PROJECT_ID=your-gcp-project-id

# 個別サービス
make deploy-librarian PROJECT_ID=your-gcp-project-id
make deploy-professor PROJECT_ID=your-gcp-project-id
make deploy-frontend  PROJECT_ID=your-gcp-project-id
```

詳細: `ops/cloudrun/README.md` / `ops/docs/CLOUD_RUN.md`

---

## デプロイ手順

### 初期セットアップ（初回のみ）

```bash
# 1. 環境変数設定
export PROJECT_ID="your-gcp-project-id"
export REGION="asia-northeast1"

# 2. 必要な API を有効化
gcloud services enable \
  run.googleapis.com \
  sqladmin.googleapis.com \
  storage.googleapis.com \
  pubsub.googleapis.com \
  secretmanager.googleapis.com \
  artifactregistry.googleapis.com \
  cloudarmor.googleapis.com

# 3. Artifact Registry リポジトリ作成
gcloud artifacts repositories create eduanima \
  --repository-format docker \
  --location ${REGION} \
  --description "EduAnimaR Docker images"

# 4. Cloud SQL インスタンス作成（pgvector 有効化）
gcloud sql instances create eduanima-pg \
  --database-version POSTGRES_18 \
  --tier db-g1-small \
  --region ${REGION}

# 5. データベース・ユーザー作成
gcloud sql databases create eduanima_professor \
  --instance eduanima-pg

gcloud sql users create eduanima \
  --instance eduanima-pg \
  --password $(openssl rand -base64 32)

# 6. Cloud Storage バケット作成
gsutil mb -l ${REGION} gs://eduanima-materials-prod

# 7. Secret Manager にシークレット登録
echo -n "your-gemini-api-key" | \
  gcloud secrets create GEMINI_API_KEY \
    --data-file=- \
    --replication-policy=automatic

# 8. Cloud Build トリガー作成
gcloud builds triggers create github \
  --name=eduanima-deploy \
  --repository=projects/$PROJECT_ID/locations/global/connections/github/repositories/eduanimaR \
  --branch-pattern=^main$ \
  --build-config=ops/cloudrun/cloudbuild.yaml
```

---

## 環境変数リファレンス

### frontend

| 変数名 | ソース | 説明 |
|--------|--------|------|
| `NODE_ENV` | 直接設定 | `production` |
| `NEXT_PUBLIC_API_BASE_URL` | ビルド ARG | ブラウザから見える API URL |
| `API_BASE_URL` | 直接設定 | SSR 用 professor の内部 URL |

### professor

| 変数名 | ソース | 説明 |
|--------|--------|------|
| `DATABASE_URL` | Secret Manager | Cloud SQL 接続文字列 |
| `GEMINI_API_KEY` | Secret Manager | Gemini API キー |
| `OBJECT_STORAGE_BACKEND` | 直接設定 | `gcs`（本番）/ `minio`（ローカル）|
| `LIBRARIAN_GRPC_ADDR` | 直接設定 | Librarian の Cloud Run URL |
| `PROFESSOR_KAFKA_WORKER_COUNT` | 直接設定 | `5`（本番推奨）|
| `PROFESSOR_EMBEDDING_CONCURRENCY` | 直接設定 | `5`（本番推奨）|
| `PROFESSOR_DB_MAX_OPEN_CONNS` | 直接設定 | `10`（concurrency=10 に合わせる）|

### librarian

| 変数名 | ソース | 説明 |
|--------|--------|------|
| `GEMINI_API_KEY` | Secret Manager | Gemini API キー |
| `LIBRARIAN_PORT` | 直接設定 | `50051` (gRPC) |
| `LIBRARIAN_MODEL_FLASH` | 直接設定 | flash モデル名 |
| `LIBRARIAN_MODEL_FLASH_LITE` | 直接設定 | flash-lite モデル名 |

---

## コスト最適化

| 施策 | 対象 | 効果 |
|------|------|------|
| `min-instances=1` | frontend, professor, librarian | コールドスタート防止 |
| `concurrency=10` | professor | Gemini API 呼び出し数を制限 |
| Cloud CDN | frontend | 静的アセットのエッジキャッシュ |
| Cloud SQL: db-g1-small | postgres | 開発初期の最小コスト |

> **推定月次コスト（低負荷時）**:  
> Cloud Run: ~$20 / Cloud SQL: ~$30 / Storage: ~$5 / ネットワーク: ~$5 = **合計約 $60/月**
