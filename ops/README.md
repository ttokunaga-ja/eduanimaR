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
- `compose.yml`（ルート直下）は `docker compose up` をプロジェクトルートから `-f` オプションなしで
  実行するための**利便性ラッパー**として例外的に存在する。
  実体は `ops/compose/docker-compose.yml`（`include:` で委譲）。
  すべての Makefile ターゲット（`make dev` 等）は `-f ops/compose/docker-compose.yml` を
  明示するため、`compose.yml` に依存しない。
