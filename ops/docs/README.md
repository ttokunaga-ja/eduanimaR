# Ops Docs Index

This directory centralizes canonical operational docs for release and incident workflows.

## Linked Docs
- `ops/docs/CLOUD_RUN.md`
- `ops/docs/CLOUD_RUN_DEPLOY.md`
- `ops/docs/TEST_ENV_SETUP.md`

## Policy
- Health/ready conventions:
	- Professor: `GET /healthz`, `GET /readyz`
	- Librarian: `GET /healthz`, `GET /readyz` on `LIBRARIAN_HEALTH_PORT`
	- Web: `GET /api/healthz`, `GET /api/readyz`
- Common observability keys: `request_id`, `trace_id`, `user_id`
- Verification command: `./scripts/verify-phase2-standards.sh` (also included in `make verify`)

## Canonical vs Service-Specific
- Cross-service operational policy (CI/CD, incident response, observability, security baseline) is canonical under `ops/docs/`.
- `apps/*/docs/05_operations/` keeps service-specific deltas only (runtime knobs, adapter limits, service-only runbooks).
- When a rule appears both in `ops/docs/` and `apps/*/docs/05_operations/`, `ops/docs/` wins.

## Migration Rule (Phase 3)
- Do not delete service docs in one shot.
- Keep existing pages and prepend "service-specific delta" notes during migration.
- Move shared policy text to `ops/docs/` first, then replace duplicated sections with links.
