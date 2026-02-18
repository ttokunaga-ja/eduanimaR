# Cloud Run デプロイアーキテクチャ

## 全体構成

```
[ユーザー / ブラウザ]
        │
        ▼
[Global Load Balancer]  ← HTTPS 終端・証明書管理
        │
        ├── /* ──────────────────── [Cloud Run: eduanima-frontend]
        │                                  │
        │                         Next.js standalone
        │                         SSR時は professor へ内部通信
        │
        └── /api/* ──────────────── [Cloud Run: eduanima-professor]
                                           │
                                 Go REST API + gRPC クライアント
                                           │
                              内部通信 (OIDC認証) ──────────────────────────────────────
                                           │
                                           ▼
                                [Cloud Run: eduanima-librarian]
                                           │
                                 Python gRPC推論サービス (LangGraph)
                                 ※ Ingress: internal のみ（外部公開なし）
```

## サービス一覧と Ingress 設定

| サービス名             | 役割                      | Ingress 設定                          | 認証                |
|----------------------|--------------------------|--------------------------------------|-------------------|
| `eduanima-frontend`  | Next.js フロントエンド     | `internal-and-cloud-load-balancing`  | 不要               |
| `eduanima-professor` | Go REST API バックエンド   | `internal-and-cloud-load-balancing`  | 不要（LB経由）     |
| `eduanima-librarian` | Python gRPC 推論サービス  | `internal`（外部公開なし）            | OIDC（サービスアカウント） |

> **セキュリティ原則**: Librarian はインターネットから直接アクセス不可。Professor の Service Account 経由でのみ呼び出せる。

---

## 本番インフラ（GCP）

### ローカル → 本番 の対応表

| ローカル (docker-compose)               | 本番 (GCP)                                                          |
|---------------------------------------|---------------------------------------------------------------------|
| `postgres` (`pgvector/pgvector:pg18`) | Cloud SQL for **PostgreSQL 18** + pgvector 0.8.1 拡張              |
| `minio` (S3互換)                       | Google Cloud Storage (GCS)                                          |
| `kafka` (`apache/kafka` KRaft)         | Cloud Pub/Sub（Phase 2 移行）                                        |
| `librarian:50051` (gRPC)              | Cloud Run (internal) + OIDC 認証                                    |

> **PostgreSQL 18 採用理由**: `uuidv7()` ネイティブ関数（拡張不要・時系列ソート可能 UUID）、`UNIQUE NULLS NOT DISTINCT`、pgvector 0.8.1 対応。Cloud SQL for PostgreSQL 18 は `asia-northeast1` リージョンで利用可能。

### 切り替えポイント（環境変数）

```bash
# Professor の環境変数で切り替え
OBJECT_STORAGE_BACKEND=gcs    # minio → GCS
DATABASE_URL=...               # localhost:5432 → Cloud SQL via Unix Socket
LIBRARIAN_GRPC_ADDR=...        # librarian:50051 → Cloud Run internal URL
```

---

## デプロイ前提条件

```bash
# 1. gcloud CLI ログイン
gcloud auth login
gcloud config set project YOUR_PROJECT_ID

# 2. Artifact Registry リポジトリ作成（初回のみ）
gcloud artifacts repositories create eduanima \
  --repository-format=docker \
  --location=asia-northeast1 \
  --description="eduanimaR container images"

# 3. Cloud Build サービスアカウントへの権限付与（初回のみ）
PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)")
SA="$PROJECT_NUMBER@cloudbuild.gserviceaccount.com"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:$SA" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:$SA" \
  --role="roles/secretmanager.secretAccessor"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:$SA" \
  --role="roles/artifactregistry.writer"

# 4. Secret Manager にシークレット登録（初回のみ）
echo -n "your-gemini-api-key" | \
  gcloud secrets create GEMINI_API_KEY \
    --data-file=- \
    --replication-policy=automatic

# 5. Cloud Build トリガー作成（初回のみ）
gcloud builds triggers create github \
  --name=eduanima-deploy \
  --repository=projects/$PROJECT_ID/locations/global/connections/github/repositories/eduanimaR \
  --branch-pattern=^main$ \
  --build-config=cloudbuild.yaml
```

---

## 手動デプロイ

```bash
# 全サービスをビルド＆デプロイ（make deploy）
make deploy PROJECT_ID=your-project-id

# 個別サービスのみデプロイ
make deploy-librarian PROJECT_ID=your-project-id
make deploy-professor PROJECT_ID=your-project-id
make deploy-frontend  PROJECT_ID=your-project-id
```

---

## Load Balancer 設定（URL マップ）

```bash
# 1. フロントエンドの NEG（Network Endpoint Group）を作成
gcloud compute network-endpoint-groups create eduanima-frontend-neg \
  --region=asia-northeast1 \
  --network-endpoint-type=SERVERLESS \
  --cloud-run-service=eduanima-frontend

# 2. Professor の NEG を作成
gcloud compute network-endpoint-groups create eduanima-professor-neg \
  --region=asia-northeast1 \
  --network-endpoint-type=SERVERLESS \
  --cloud-run-service=eduanima-professor

# 3. バックエンドサービス作成
gcloud compute backend-services create eduanima-frontend-backend \
  --global
gcloud compute backend-services add-backend eduanima-frontend-backend \
  --global \
  --network-endpoint-group=eduanima-frontend-neg \
  --network-endpoint-group-region=asia-northeast1

gcloud compute backend-services create eduanima-professor-backend \
  --global
gcloud compute backend-services add-backend eduanima-professor-backend \
  --global \
  --network-endpoint-group=eduanima-professor-neg \
  --network-endpoint-group-region=asia-northeast1

# 4. URL マップ（パスベースルーティング）
gcloud compute url-maps create eduanima-url-map \
  --default-service=eduanima-frontend-backend

gcloud compute url-maps import eduanima-url-map --global << 'EOF'
defaultService: global/backendServices/eduanima-frontend-backend
hostRules:
  - hosts: ["*"]
    pathMatcher: main
pathMatchers:
  - name: main
    defaultService: global/backendServices/eduanima-frontend-backend
    pathRules:
      - paths: ["/api/*"]
        service: global/backendServices/eduanima-professor-backend
EOF
```

---

## サービス間認証（Professor → Librarian）

Cloud Run の Librarian は `--ingress=internal` のため、Professor は OIDC トークンを付けて呼び出す必要があります。

```go
// Professor 側（Go）での gRPC 接続例
// google.golang.org/grpc/credentials/oauth を使用
import "google.golang.org/grpc/credentials/oauth"

audience := "https://eduanima-librarian-XXXXXX-an.a.run.app"
creds, err := oauth.NewServiceAccountCredentials(audience)
conn, err := grpc.Dial(librarianAddr,
    grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, "")),
    grpc.WithPerRPCCredentials(creds),
)
```

> ローカル開発（docker-compose）では認証不要（同一ネットワーク内の名前解決）。

---

## デプロイフロー（CI/CD）

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

> ビルドキャッシュ（`--cache-from :latest`）を活用して CI 時間を短縮。

---

## 注意事項

- `NEXT_PUBLIC_API_BASE_URL` は Next.js ビルド時に埋め込まれるため、**デプロイ前に正しい URL に設定すること**
- Cloud SQL との接続は Unix ソケット（`/cloudsql/INSTANCE_CONNECTION_NAME`）を使用する
- Kafka（ローカル）→ Cloud Pub/Sub（本番）の切り替えは Phase 2 で対応（**本番では Kafka を使わない**）
- **本番移行時の Kafka 設定チェックリスト**（万が一本番でも Kafka を使う場合）:
  - `KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR` をブローカー数に合わせて設定（デフォルト `3`、単一ブローカーでは `1`）
  - `KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR` も同様に設定
  - 開発用 `docker-compose.yml` の `replication.factor=1` 設定は**本番向けではない**
- Librarian は gRPC（HTTP/2）を使用するため、Cloud Run の HTTP/2 を有効化すること
