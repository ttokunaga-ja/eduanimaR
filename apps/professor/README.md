# eduanima-professor

Go backend service for API, data access, and integration orchestration.

## Responsibility
- Exposes REST API to Web/clients.
- Owns DB schema, storage integration, and ingestion pipeline orchestration.
- Calls Librarian over gRPC for reasoning support.

## Contracts
- OpenAPI SSOT: `../../contracts/openapi/professor.yaml`
- Proto SSOT: `../../contracts/proto/librarian/v1/librarian.proto`
- Service docs: `docs/README.md`

## Common Commands
```bash
make generate         # proto + sqlc codegen
make build            # compile binary
make run              # run API locally
make lint             # golangci-lint (docker fallback)
make test-unit        # unit tests
make test-contract    # contract tests
```

## Test Topology
- Unit tests: `internal/usecases`, `internal/domain`
- Contract tests: `tests/contract`
- Integration tests: `tests/integration` (scaffold)

## Health And Observability
- Health endpoints: `GET /healthz`, `GET /readyz`
- Common log keys: `request_id`, `trace_id`, `user_id`
