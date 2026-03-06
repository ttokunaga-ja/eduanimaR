#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

# 1) Ops assets are centralized under ops/
required_ops_files=(
  "ops/compose/docker-compose.yml"
  "ops/compose/docker-compose.prod.yml"
  "ops/cloudrun/cloudbuild.yaml"
  "ops/docs/CLOUD_RUN.md"
)
for f in "${required_ops_files[@]}"; do
  [[ -f "$f" ]] || { echo "❌ missing canonical ops file: $f"; exit 1; }
done

# Root-level legacy runtime files must not exist.
legacy_root_files=(
  "docker-compose.yml"
  "docker-compose.prod.yml"
  "cloudbuild.yaml"
  "CLOUD_RUN.md"
)
for f in "${legacy_root_files[@]}"; do
  [[ ! -f "$f" ]] || { echo "❌ legacy root runtime file must not exist: $f"; exit 1; }
done

# 2) No stale references to removed root runtime files.
stale_refs=$(grep -RInE "(^|[^/[:alnum:]_.-])docker-compose\.yml([^/[:alnum:]_.-]|$)|(^|[^/[:alnum:]_.-])docker-compose\.prod\.yml([^/[:alnum:]_.-]|$)|(^|[^/[:alnum:]_.-])cloudbuild\.yaml([^/[:alnum:]_.-]|$)|(^|[^/[:alnum:]_.-])CLOUD_RUN\.md([^/[:alnum:]_.-]|$)" . --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=.venv || true)
stale_refs=$(printf '%s\n' "$stale_refs" | grep -v 'scripts/verify-phase2-standards.sh' || true)
if [[ -n "$stale_refs" ]]; then
  echo "❌ stale references to removed root runtime files found:"
  echo "$stale_refs"
  exit 1
fi

# 3) Health/readiness endpoints are present per service.
grep -q 'e.GET("/healthz"' apps/professor/cmd/professor/main.go || { echo "❌ professor /healthz route missing"; exit 1; }
grep -q 'e.GET("/readyz"' apps/professor/cmd/professor/main.go || { echo "❌ professor /readyz route missing"; exit 1; }
grep -q '"/healthz"' apps/librarian/src/librarian/main.py || { echo "❌ librarian /healthz handler missing"; exit 1; }
grep -q '"/readyz"' apps/librarian/src/librarian/main.py || { echo "❌ librarian /readyz handler missing"; exit 1; }
[[ -f apps/web/src/app/api/healthz/route.ts ]] || { echo "❌ web /api/healthz route missing"; exit 1; }
[[ -f apps/web/src/app/api/readyz/route.ts ]] || { echo "❌ web /api/readyz route missing"; exit 1; }

# 4) Common observability keys exist in all services.
grep -q '"request_id"' apps/professor/internal/adapters/http/middleware/requestlog.go || { echo "❌ professor request_id key missing"; exit 1; }
grep -q '"trace_id"' apps/professor/internal/adapters/http/middleware/requestlog.go || { echo "❌ professor trace_id key missing"; exit 1; }
grep -q '"user_id"' apps/professor/internal/adapters/http/middleware/requestlog.go || { echo "❌ professor user_id key missing"; exit 1; }
grep -q '"request_id"' apps/librarian/src/librarian/server.py || { echo "❌ librarian request_id key missing"; exit 1; }
grep -q '"trace_id"' apps/librarian/src/librarian/server.py || { echo "❌ librarian trace_id key missing"; exit 1; }
grep -q '"user_id"' apps/librarian/src/librarian/server.py || { echo "❌ librarian user_id key missing"; exit 1; }
grep -q 'request_id' apps/web/src/app/api/healthz/route.ts || { echo "❌ web request_id key missing"; exit 1; }
grep -q 'trace_id' apps/web/src/app/api/healthz/route.ts || { echo "❌ web trace_id key missing"; exit 1; }
grep -q 'user_id' apps/web/src/app/api/healthz/route.ts || { echo "❌ web user_id key missing"; exit 1; }

echo "✅ phase2 standards checks passed"
