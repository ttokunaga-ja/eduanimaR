# Ops Portal

This directory is the operational entry point for deploy/run/incident workflows.

## Structure
- `ops/compose/`: local and production compose orchestration references.
- `ops/cloudrun/`: Cloud Run and Cloud Build deployment references.
- `ops/docs/`: operational runbooks and migration stubs.

## Canonical Runtime Files
- `ops/compose/docker-compose.yml`
- `ops/compose/docker-compose.prod.yml`
- `ops/cloudrun/cloudbuild.yaml`
- `ops/docs/CLOUD_RUN.md`

## Notes
- Runtime/deploy entrypoints are standardized under `ops/`.
- Root-level runtime manifests are intentionally not used.
