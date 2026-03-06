# Cloud Run Operations Index

## Canonical Files
- `cloudbuild.yaml`: build/deploy pipeline definition.
- `CLOUD_RUN.md`: architecture and deployment details.
- `docs/CLOUD_RUN_DEPLOY.md`: additional deployment architecture notes.

## Common Commands
```bash
make deploy PROJECT_ID=<gcp-project>
make deploy-professor PROJECT_ID=<gcp-project>
make deploy-librarian PROJECT_ID=<gcp-project>
make deploy-frontend PROJECT_ID=<gcp-project>
```

## Migration Plan
- Keep deployment manifests in existing locations during Phase 1.
- Maintain this index as the top-level Cloud Run entrypoint.
