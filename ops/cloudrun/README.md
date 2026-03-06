# Cloud Run Operations Index

## Canonical Files
- `ops/cloudrun/cloudbuild.yaml`: build/deploy pipeline definition.
- `ops/docs/CLOUD_RUN.md`: architecture and deployment details.
- `ops/docs/CLOUD_RUN_DEPLOY.md`: additional deployment architecture notes.

## Common Commands
```bash
make deploy PROJECT_ID=<gcp-project>
make deploy-professor PROJECT_ID=<gcp-project>
make deploy-librarian PROJECT_ID=<gcp-project>
make deploy-frontend PROJECT_ID=<gcp-project>
```

## Policy
- Use `ops/cloudrun/cloudbuild.yaml` for build/deploy pipeline execution.
- Maintain this index as the top-level Cloud Run entrypoint.
