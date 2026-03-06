# Skill: Deploy to GCP (Cloud Run + Docker + GitHub Actions)

## Purpose
Ship services to production reproducibly with:
- **Docker image** as the deployment unit
- **Cloud Run** as the runtime
- **GitHub Actions** as the CI/CD runner
- **.env (local dev)** / **GitHub Secrets + GCP Secret Manager (release)** for configuration

## What this doc gives you
This Skill includes a **copy-pasteable bootstrap**:
- Enable required GCP APIs
- Create an Artifact Registry repo (Docker)
- Configure **Workload Identity Federation (WIF)** for GitHub Actions (no key JSON)
- Set minimal IAM roles
- Create secrets and map them into Cloud Run
- First deploy + rollback via revisions

## SSOT
- CI/CD: `docs/05_operations/CI_CD.md`
- Release & deploy: `docs/05_operations/RELEASE_DEPLOY.md`
- Secrets & key management: `docs/05_operations/SECRETS_KEY_MANAGEMENT.md`
- Supply chain: `docs/skills/SKILL_SUPPLY_CHAIN_SLSA_SBOM.md`

## Rules (non-negotiable)
- **Cloud Run + Docker + GitHub Actions only** for deploy.
- **Local configuration** is managed via **`.env`** (not committed).
- **Release/production configuration** is managed via:
  - **GitHub Secrets**: CI/CD-time values (e.g. GCP project id, WIF provider, service account email, registry/repo identifiers)
  - **GCP Secret Manager**: runtime application configuration (DB URLs, API keys, OAuth secrets, encryption keys, and other env vars)
- **Do not use any external “environment variable management” services** besides GitHub Secrets and GCP Secret Manager.
  - 明記: **GitHub Secrets + GCP Secret Manager 以外の外部依存の変数管理サービスは使用しない**
  - Explicitly forbidden examples: Doppler, 1Password CLI, Vault SaaS, Parameter Store-like third parties, custom hosted secret UIs.
- **No service account key JSON** (long-lived keys) in repo, developer machines, or GitHub Secrets.
  - Use **Workload Identity Federation (WIF)** from GitHub Actions.

## Safe defaults
- Prefer **Artifact Registry** for container images.
- Use **immutable image tags** for releases (git SHA, semver), and deploy by tag.
- Inject configuration via Cloud Run environment variables **sourced from Secret Manager**.
- Keep “non-secret config” minimal; when in doubt, treat as config and store in Secret Manager to avoid drift.
- Enable structured logs and trace propagation; never log secret values.

## Secret & env-var policy (must match all environments)
### Local (developer machine)
- Store configuration in `.env`.
- `.env` MUST be in `.gitignore`.
- The application loads env vars from the OS environment; `.env` is only a convenience for local dev.

### Release / Production
- Runtime env vars are injected from **GCP Secret Manager**.
- GitHub Actions uses **GitHub Secrets** only for CI/CD wiring (authentication, project/repo identifiers), not as the runtime config source.
- Cloud Run MUST reference secrets (env var mapping) instead of hardcoding values in workflows.

## GitHub Actions (reference workflow outline)
Minimal outline (pseudo-config; adapt names to your repo):

- Trigger: on push to `main` and on tags (release)
- Auth to GCP: `google-github-actions/auth` via **WIF**
- Build/push image: Docker Buildx to Artifact Registry
- Deploy: `gcloud run deploy`

Example steps (illustrative):

```yaml
permissions:
  id-token: write
  contents: read

steps:
  - uses: actions/checkout@v4

  - uses: google-github-actions/auth@v2
    with:
      workload_identity_provider: ${{ secrets.GCP_WIF_PROVIDER }}
      service_account: ${{ secrets.GCP_SERVICE_ACCOUNT_EMAIL }}

  - uses: google-github-actions/setup-gcloud@v2

  - run: |
      gcloud auth configure-docker ${REGION}-docker.pkg.dev

  - uses: docker/setup-buildx-action@v3

  - uses: docker/build-push-action@v6
    with:
      context: .
      push: true
      tags: ${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/${IMAGE}:${GITHUB_SHA}

  - run: |
      gcloud run deploy ${SERVICE_NAME} \
        --image=${REGION}-docker.pkg.dev/${PROJECT_ID}/${REPO}/${IMAGE}:${GITHUB_SHA} \
        --region=${REGION} \
        --platform=managed \
        --allow-unauthenticated=false \
        --set-secrets=DATABASE_URL=DATABASE_URL:latest,JWT_SIGNING_KEY=JWT_SIGNING_KEY:latest
```

Notes:
- Prefer `--allow-unauthenticated=false` by default; expose via Gateway/IAP/Load Balancer.
- `--set-secrets` supports `ENV_VAR=SECRET_NAME:version`. Use explicit versions for strict rollouts.

---

## Bootstrap (copy-paste): GCP setup for Cloud Run + Artifact Registry + WIF

### Prerequisites
- You have a GCP project (or create one) and billing is enabled.
- You have `gcloud` installed and authenticated locally.
- Your repo is on GitHub and you know `<owner>/<repo>`.

### 0) Variables (edit these once)

```bash
# Required
export PROJECT_ID="YOUR_GCP_PROJECT_ID"
export REGION="asia-northeast1"              # choose one region and standardize
export SERVICE_NAME="your-service"           # Cloud Run service name
export IMAGE_NAME="your-service"             # container image name
export AR_REPO="services"                    # Artifact Registry repository name

# GitHub
export GITHUB_OWNER="YOUR_GH_OWNER"
export GITHUB_REPO="YOUR_GH_REPO"

# Identities
export SA_DEPLOY_NAME="gh-actions-deployer"  # used by GitHub Actions via WIF
export SA_RUNTIME_NAME="cloudrun-runtime"    # used by Cloud Run at runtime

# WIF
export WIF_POOL="github-pool"
export WIF_PROVIDER="github-provider"
```

Set the active project:

```bash
gcloud config set project "$PROJECT_ID"
```

### 1) Enable required APIs

```bash
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  iam.googleapis.com \
  iamcredentials.googleapis.com \
  sts.googleapis.com \
  secretmanager.googleapis.com
```

### 2) Create Artifact Registry (Docker)

Create a Docker repo (once per region/repo):

```bash
gcloud artifacts repositories create "$AR_REPO" \
  --repository-format=docker \
  --location="$REGION" \
  --description="Container images for services"
```

Configure Docker auth (local sanity check):

```bash
gcloud auth configure-docker "${REGION}-docker.pkg.dev"
```

### 3) Create service accounts (deploy + runtime)

```bash
gcloud iam service-accounts create "$SA_DEPLOY_NAME" \
  --display-name="GitHub Actions deployer (WIF)"

gcloud iam service-accounts create "$SA_RUNTIME_NAME" \
  --display-name="Cloud Run runtime identity"

export SA_DEPLOY_EMAIL="$SA_DEPLOY_NAME@$PROJECT_ID.iam.gserviceaccount.com"
export SA_RUNTIME_EMAIL="$SA_RUNTIME_NAME@$PROJECT_ID.iam.gserviceaccount.com"
```

### 4) IAM roles (minimal)

Grant the deployer the ability to deploy Cloud Run and push images:

```bash
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA_DEPLOY_EMAIL" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA_DEPLOY_EMAIL" \
  --role="roles/artifactregistry.writer"
```

Allow the deployer to set the runtime service account on Cloud Run deploys:

```bash
gcloud iam service-accounts add-iam-policy-binding "$SA_RUNTIME_EMAIL" \
  --member="serviceAccount:$SA_DEPLOY_EMAIL" \
  --role="roles/iam.serviceAccountUser"
```

Grant the runtime identity access to secrets at runtime:

```bash
gcloud projects add-iam-policy-binding "$PROJECT_ID" \
  --member="serviceAccount:$SA_RUNTIME_EMAIL" \
  --role="roles/secretmanager.secretAccessor"
```

### 5) Workload Identity Federation (GitHub Actions → GCP)

Create a Workload Identity Pool:

```bash
gcloud iam workload-identity-pools create "$WIF_POOL" \
  --location="global" \
  --display-name="GitHub Actions"
```

Create an OIDC provider for GitHub:

```bash
gcloud iam workload-identity-pools providers create-oidc "$WIF_PROVIDER" \
  --location="global" \
  --workload-identity-pool="$WIF_POOL" \
  --display-name="GitHub OIDC" \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref" \
  --attribute-condition="attribute.repository=='${GITHUB_OWNER}/${GITHUB_REPO}'"
```

Allow GitHub identities in this repo to impersonate the deployer service account:

```bash
export PROJECT_NUMBER="$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')"

gcloud iam service-accounts add-iam-policy-binding "$SA_DEPLOY_EMAIL" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${WIF_POOL}/attribute.repository/${GITHUB_OWNER}/${GITHUB_REPO}"
```

Compute these values for GitHub Secrets:

```bash
export WIF_PROVIDER_RESOURCE="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${WIF_POOL}/providers/${WIF_PROVIDER}"
echo "WIF provider: $WIF_PROVIDER_RESOURCE"
echo "Deployer SA:  $SA_DEPLOY_EMAIL"
```

---

## Bootstrap (copy-paste): Secret Manager + Cloud Run deploy + rollback

### 6) Create secrets (example)

Create secrets (names are examples; standardize naming in your org):

```bash
gcloud secrets create GEMINI_API_KEY --replication-policy="automatic"
gcloud secrets create JWT_SIGNING_KEY --replication-policy="automatic"
```

Add initial secret versions (avoid leaking secrets into shell history in real ops; this is for bootstrap):

```bash
printf '%s' 'YOUR_GEMINI_API_KEY' | \
  gcloud secrets versions add GEMINI_API_KEY --data-file=-

openssl rand -base64 48 | \
  gcloud secrets versions add JWT_SIGNING_KEY --data-file=-
```

### 7) First deploy (from local, sanity check)

Build & push one image locally (optional sanity check; CI should be the normal path):

```bash
export IMAGE_URI="${REGION}-docker.pkg.dev/${PROJECT_ID}/${AR_REPO}/${IMAGE_NAME}:local-$(date +%Y%m%d%H%M%S)"

docker build -t "$IMAGE_URI" .
docker push "$IMAGE_URI"
```

Deploy to Cloud Run using the runtime SA and Secret Manager mappings:

```bash
gcloud run deploy "$SERVICE_NAME" \
  --image="$IMAGE_URI" \
  --region="$REGION" \
  --platform="managed" \
  --service-account="$SA_RUNTIME_EMAIL" \
  --allow-unauthenticated=false \
  --set-secrets="GEMINI_API_KEY=GEMINI_API_KEY:latest,JWT_SIGNING_KEY=JWT_SIGNING_KEY:latest"
```

### 8) Rollback (by revision)

List revisions:

```bash
gcloud run revisions list --service "$SERVICE_NAME" --region "$REGION"
```

Shift 100% traffic back to a previous revision:

```bash
gcloud run services update-traffic "$SERVICE_NAME" \
  --region "$REGION" \
  --to-revisions "REVISION_NAME=100"
```

If you deploy by immutable image tag (recommended), you can also roll back by redeploying a known-good tag.

---

## GitHub Actions: minimal working workflow (copy-paste)

Create `.github/workflows/deploy-cloud-run.yml`:

```yaml
name: deploy-cloud-run

on:
  push:
    branches: ["main"]

permissions:
  id-token: write
  contents: read

env:
  REGION: asia-northeast1
  AR_REPO: services
  SERVICE_NAME: your-service
  IMAGE_NAME: your-service

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - id: auth
        uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ secrets.GCP_WIF_PROVIDER }}
          service_account: ${{ secrets.GCP_SERVICE_ACCOUNT_EMAIL }}

      - uses: google-github-actions/setup-gcloud@v2

      - name: Configure docker auth
        run: gcloud auth configure-docker "${REGION}-docker.pkg.dev" --quiet

      - uses: docker/setup-buildx-action@v3

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: |
            ${REGION}-docker.pkg.dev/${{ secrets.GCP_PROJECT_ID }}/${AR_REPO}/${IMAGE_NAME}:${{ github.sha }}

      - name: Deploy
        run: |
          gcloud run deploy "${SERVICE_NAME}" \
            --image="${REGION}-docker.pkg.dev/${{ secrets.GCP_PROJECT_ID }}/${AR_REPO}/${IMAGE_NAME}:${{ github.sha }}" \
            --region="${REGION}" \
            --platform=managed \
            --allow-unauthenticated=false \
            --service-account="${{ secrets.CLOUD_RUN_RUNTIME_SA_EMAIL }}" \
            --set-secrets="DATABASE_URL=DATABASE_URL:latest,JWT_SIGNING_KEY=JWT_SIGNING_KEY:latest"
```

Required GitHub Secrets:
- `GCP_PROJECT_ID`
- `GCP_WIF_PROVIDER` (example: `projects/123456789012/locations/global/workloadIdentityPools/github-pool/providers/github-provider`)
- `GCP_SERVICE_ACCOUNT_EMAIL` (the deployer SA email)
- `CLOUD_RUN_RUNTIME_SA_EMAIL` (the runtime SA email)

---

## Decision checklist (project must choose once)
- Region (`REGION`) and Artifact Registry location
- Naming: `SERVICE_NAME`, `AR_REPO`, secret names
- Runtime identity: per-service SA vs shared SA
- Exposure: `--allow-unauthenticated` policy (default false) and ingress path (Gateway/IAP/LB)
- Rollout policy: tag strategy (git SHA vs semver) and rollback method (revision vs tag)

## Docker (safe defaults)
- Multi-stage build.
- Run as non-root.
- `PORT` from Cloud Run is honored.
- No `.env` copied into the image.

## “Banned tools” compliance (deployment/config-related)
- Config libraries: **viper** is forbidden. Use simple `os.Getenv` + explicit parsing/validation.
- Secret managers: **Doppler** (and similar) is forbidden; use **GitHub Secrets + GCP Secret Manager** only.
- Logging: avoid `fmt.Println` / `log.Println` / logrus; use `log/slog` (Go 1.21+).

## Checklist
- Cloud Run deploy uses an image from Artifact Registry.
- GitHub Actions auth uses WIF (OIDC), not a service account JSON key.
- Local config comes from `.env` and is not committed.
- Production config is stored in GCP Secret Manager and injected into Cloud Run.
- No third-party env-var management service is introduced.
- Release uses immutable image tags and can be rolled back by tag.
