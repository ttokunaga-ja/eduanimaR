# Cloud Run デプロイアーキテクチャ

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
│  │  │  PostgreSQL 17 │      │             │ Cloud Run internal  │ │   │
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

```bash
gcloud run deploy eduanima-frontend \
  --image asia-northeast1-docker.pkg.dev/${PROJECT_ID}/eduanima/frontend:${TAG} \
  --region asia-northeast1 \
  --platform managed \
  --allow-unauthenticated \
  --min-instances 1 \
  --max-instances 10 \
  --memory 512Mi \
  --cpu 1 \
  --concurrency 80 \
  --port 3000 \
  --set-env-vars "NODE_ENV=production,NEXT_TELEMETRY_DISABLED=1" \
  --set-secrets "API_BASE_URL=eduanima-professor-url:latest" \
  --service-account eduanima-frontend-sa@${PROJECT_ID}.iam.gserviceaccount.com
```

### 2. eduanima-professor（Go 1.25）

| 項目 | 設定値 |
|------|--------|
| イメージ | `asia-northeast1-docker.pkg.dev/[PROJECT]/eduanima/professor:$TAG` |
| アクセス | **Internal + Cloud LB** (frontend からのみ) |
| リージョン | `asia-northeast1` |
| 最小インスタンス | 1 |
| 最大インスタンス | 20 |
| CPU | 2 vCPU |
| メモリ | 1Gi |
| 同時接続数 | 100 |
| ポート | 8080 |

```bash
gcloud run deploy eduanima-professor \
  --image asia-northeast1-docker.pkg.dev/${PROJECT_ID}/eduanima/professor:${TAG} \
  --region asia-northeast1 \
  --platform managed \
  --no-allow-unauthenticated \
  --ingress internal-and-cloud-load-balancing \
  --min-instances 1 \
  --max-instances 20 \
  --memory 1Gi \
  --cpu 2 \
  --concurrency 100 \
  --port 8080 \
  --set-secrets \
    "GEMINI_API_KEY=gemini-api-key:latest,\
     DATABASE_URL=professor-database-url:latest,\
     MINIO_ACCESS_KEY=gcs-hmac-access-key:latest,\
     MINIO_SECRET_KEY=gcs-hmac-secret-key:latest" \
  --set-env-vars \
    "OBJECT_STORAGE_BACKEND=gcs,\
     KAFKA_BROKERS=pubsub,\
     LIBRARIAN_GRPC_ADDR=eduanima-librarian.internal:50051,\
     PORT=8080,\
     LOG_LEVEL=info" \
  --service-account eduanima-professor-sa@${PROJECT_ID}.iam.gserviceaccount.com \
  --add-cloudsql-instances ${PROJECT_ID}:asia-northeast1:eduanima-pg
```

### 3. eduanima-librarian（Python 3.12）

| 項目 | 設定値 |
|------|--------|
| イメージ | `asia-northeast1-docker.pkg.dev/[PROJECT]/eduanima/librarian:$TAG` |
| アクセス | **Internal Only** (professor からのみ) |
| リージョン | `asia-northeast1` |
| 最小インスタンス | 0（コスト最適化）|
| 最大インスタンス | 5 |
| CPU | 2 vCPU |
| メモリ | 2Gi |
| 同時接続数 | 10 |
| ポート | 50051 (gRPC) |

```bash
gcloud run deploy eduanima-librarian \
  --image asia-northeast1-docker.pkg.dev/${PROJECT_ID}/eduanima/librarian:${TAG} \
  --region asia-northeast1 \
  --platform managed \
  --no-allow-unauthenticated \
  --ingress internal \
  --min-instances 0 \
  --max-instances 5 \
  --memory 2Gi \
  --cpu 2 \
  --concurrency 10 \
  --port 50051 \
  --use-http2 \
  --set-secrets "GEMINI_API_KEY=gemini-api-key:latest" \
  --set-env-vars \
    "GRPC_PORT=50051,\
     LOG_LEVEL=info" \
  --service-account eduanima-librarian-sa@${PROJECT_ID}.iam.gserviceaccount.com
```

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

### Cloud Run サービス間通信（gRPC）

```
professor → librarian の内部 gRPC 通信:

LIBRARIAN_GRPC_ADDR = https://eduanima-librarian-xxxx-an.a.run.app

# Cloud Run の内部サービス間通信は HTTPS（gRPC over HTTP/2）
# Cloud Run の --use-http2 フラグで gRPC を有効化
```

---

## マネージドサービス接続

### Phase 1 → Phase 2 移行マッピング

| ローカル（Phase 1） | Cloud（Phase 2） | 変更箇所 |
|---------------------|-----------------|---------|
| PostgreSQL + pgvector | Cloud SQL for PostgreSQL 17 + pgvector | `DATABASE_URL` のみ変更 |
| MinIO | Cloud Storage + HMAC 認証 | `OBJECT_STORAGE_BACKEND=gcs` |
| Kafka (KRaft) | Cloud Pub/Sub | Kafka クライアントを Pub/Sub SDK に置換 |

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
  gcloud secrets create gemini-api-key \
    --data-file=- \
    --replication-policy=regional \
    --locations=asia-northeast1

# シークレット一覧
gcloud secrets create gemini-api-key        # Gemini API Key
gcloud secrets create professor-database-url # Cloud SQL 接続文字列
gcloud secrets create gcs-hmac-access-key   # GCS HMAC アクセスキー
gcloud secrets create gcs-hmac-secret-key   # GCS HMAC シークレット
gcloud secrets create eduanima-professor-url # Professor の Cloud Run URL
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

### GitHub Actions ワークフロー

```yaml
# .github/workflows/deploy.yml
name: Deploy to Cloud Run

on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      service:
        description: 'デプロイするサービス'
        required: true
        type: choice
        options: [all, frontend, professor, librarian]

env:
  PROJECT_ID: ${{ secrets.GCP_PROJECT_ID }}
  REGION: asia-northeast1
  REGISTRY: asia-northeast1-docker.pkg.dev

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write   # Workload Identity Federation

    steps:
      - uses: actions/checkout@v4

      - name: Google Auth（Workload Identity）
        uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ secrets.WIF_PROVIDER }}
          service_account: ${{ secrets.WIF_SERVICE_ACCOUNT }}

      - name: Docker 認証
        run: gcloud auth configure-docker ${{ env.REGISTRY }} --quiet

      - name: イメージタグ生成
        id: tag
        run: echo "TAG=$(git rev-parse --short HEAD)" >> $GITHUB_OUTPUT

      # ─── Frontend ───────────────────────────────────
      - name: Frontend ビルド & プッシュ
        if: inputs.service == 'all' || inputs.service == 'frontend' || github.event_name == 'push'
        run: |
          docker build \
            --build-arg NEXT_PUBLIC_API_BASE_URL="https://api.eduanima.app" \
            -t $REGISTRY/$PROJECT_ID/eduanima/frontend:${{ steps.tag.outputs.TAG }} \
            -t $REGISTRY/$PROJECT_ID/eduanima/frontend:latest \
            ./eduanimaR
          docker push $REGISTRY/$PROJECT_ID/eduanima/frontend:${{ steps.tag.outputs.TAG }}
          docker push $REGISTRY/$PROJECT_ID/eduanima/frontend:latest

      - name: Frontend デプロイ
        if: inputs.service == 'all' || inputs.service == 'frontend' || github.event_name == 'push'
        uses: google-github-actions/deploy-cloudrun@v2
        with:
          service: eduanima-frontend
          region: ${{ env.REGION }}
          image: ${{ env.REGISTRY }}/${{ env.PROJECT_ID }}/eduanima/frontend:${{ steps.tag.outputs.TAG }}

      # ─── Professor ──────────────────────────────────
      - name: Professor ビルド & プッシュ
        if: inputs.service == 'all' || inputs.service == 'professor' || github.event_name == 'push'
        run: |
          docker build \
            -t $REGISTRY/$PROJECT_ID/eduanima/professor:${{ steps.tag.outputs.TAG }} \
            -t $REGISTRY/$PROJECT_ID/eduanima/professor:latest \
            --target production \
            ./eduanimaR_Professor
          docker push $REGISTRY/$PROJECT_ID/eduanima/professor:${{ steps.tag.outputs.TAG }}
          docker push $REGISTRY/$PROJECT_ID/eduanima/professor:latest

      - name: Professor デプロイ
        if: inputs.service == 'all' || inputs.service == 'professor' || github.event_name == 'push'
        uses: google-github-actions/deploy-cloudrun@v2
        with:
          service: eduanima-professor
          region: ${{ env.REGION }}
          image: ${{ env.REGISTRY }}/${{ env.PROJECT_ID }}/eduanima/professor:${{ steps.tag.outputs.TAG }}

      # ─── Librarian ──────────────────────────────────
      - name: Librarian ビルド & プッシュ
        if: inputs.service == 'all' || inputs.service == 'librarian' || github.event_name == 'push'
        run: |
          docker build \
            -t $REGISTRY/$PROJECT_ID/eduanima/librarian:${{ steps.tag.outputs.TAG }} \
            -t $REGISTRY/$PROJECT_ID/eduanima/librarian:latest \
            ./eduanimaR_Librarian
          docker push $REGISTRY/$PROJECT_ID/eduanima/librarian:${{ steps.tag.outputs.TAG }}
          docker push $REGISTRY/$PROJECT_ID/eduanima/librarian:latest

      - name: Librarian デプロイ
        if: inputs.service == 'all' || inputs.service == 'librarian' || github.event_name == 'push'
        uses: google-github-actions/deploy-cloudrun@v2
        with:
          service: eduanima-librarian
          region: ${{ env.REGION }}
          image: ${{ env.REGISTRY }}/${{ env.PROJECT_ID }}/eduanima/librarian:${{ steps.tag.outputs.TAG }}
```

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
  --database-version POSTGRES_17 \
  --tier db-g1-small \
  --region ${REGION} \
  --database-flags cloudsql.enable_pgvector=on

# 5. データベース・ユーザー作成
gcloud sql databases create eduanima_professor \
  --instance eduanima-pg

gcloud sql users create eduanima \
  --instance eduanima-pg \
  --password $(openssl rand -base64 32)

# 6. Cloud Storage バケット作成
gsutil mb -l ${REGION} gs://eduanima-materials-prod
gsutil iam ch serviceAccount:eduanima-professor-sa@${PROJECT_ID}.iam.gserviceaccount.com:objectAdmin \
  gs://eduanima-materials-prod

# 7. Pub/Sub トピック作成（Kafka 代替）
gcloud pubsub topics create eduanima.ingest.jobs
gcloud pubsub subscriptions create eduanima-professor-ingest \
  --topic eduanima.ingest.jobs \
  --ack-deadline 60
```

### 通常デプロイ

```bash
# GitHub Actions の main ブランチプッシュで自動デプロイ
# または手動トリガー:
gh workflow run deploy.yml --field service=all
```

---

## 環境変数リファレンス

### frontend

| 変数名 | ソース | 説明 |
|--------|--------|------|
| `NODE_ENV` | 直接設定 | `production` |
| `NEXT_PUBLIC_API_BASE_URL` | ビルド ARG | ブラウザから見える API URL（例: `https://api.eduanima.app`）|
| `API_BASE_URL` | Secret Manager | SSR 用 professor の内部 URL |

### professor

| 変数名 | ソース | 説明 |
|--------|--------|------|
| `DATABASE_URL` | Secret Manager | Cloud SQL 接続文字列 |
| `GEMINI_API_KEY` | Secret Manager | Gemini API キー |
| `OBJECT_STORAGE_BACKEND` | 直接設定 | `gcs`（本番）/ `minio`（ローカル）|
| `KAFKA_BROKERS` | 直接設定 | Phase 2 では `pubsub` |
| `LIBRARIAN_GRPC_ADDR` | 直接設定 | Librarian の Cloud Run URL |

### librarian

| 変数名 | ソース | 説明 |
|--------|--------|------|
| `GEMINI_API_KEY` | Secret Manager | Gemini API キー |
| `GRPC_PORT` | 直接設定 | `50051` |

---

## コスト最適化

| 施策 | 対象 | 効果 |
|------|------|------|
| `min-instances=0` | librarian | アイドル時の課金ゼロ |
| `min-instances=1` | frontend, professor | コールドスタート防止（UX優先）|
| Cloud CDN | frontend | 静的アセットのエッジキャッシュ |
| Committed Use Discount | professor | 年間コミットで最大 57% 削減 |
| Cloud SQL: db-g1-small | postgres | 開発初期の最小コスト |

> **推定月次コスト（低負荷時）**:  
> Cloud Run: ~$20 / Cloud SQL: ~$30 / Storage: ~$5 / ネットワーク: ~$5 = **合計約 $60/月**
