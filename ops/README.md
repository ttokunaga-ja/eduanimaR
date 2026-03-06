# Ops Portal

This directory is the operational entry point for deploy/run/incident workflows.

## Structure
- `ops/compose/`: local and production compose orchestration references.
- `ops/cloudrun/`: Cloud Run and Cloud Build deployment references.
- `ops/docs/`: operational runbooks and migration stubs.

## Canonical Runtime Files (current locations)
- `docker-compose.yml`
- `docker-compose.prod.yml`
- `cloudbuild.yaml`
- `CLOUD_RUN.md`

## Notes
- Runtime-critical files are not moved in one shot.
- This portal consolidates navigation first, then gradual migration can follow.
