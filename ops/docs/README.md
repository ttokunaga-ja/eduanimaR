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
