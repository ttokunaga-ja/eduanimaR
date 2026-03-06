# eduanima-librarian

Python gRPC microservice for reasoning/search strategy.

## Responsibility
- Decides search strategy and stop conditions.
- Returns structured evidence candidates to Professor.
- Does not own database, indexing, or final response rendering.

## Contracts
- Proto SSOT: `../../contracts/proto/librarian/v1/librarian.proto`
- Service docs: `docs/README.md`

## Common Commands
```bash
make install          # create/repair venv and install deps
make proto            # generate proto stubs (gen/proto/python)
make run              # run service
make lint             # ruff check + format check
make typecheck        # mypy
make test             # unit tests
```

## Test Topology
- `tests/unit/`
- `tests/integration/` (scaffold)
- `tests/contract/` (scaffold)

## Health And Observability
- Health endpoints: `GET /healthz`, `GET /readyz` on `LIBRARIAN_HEALTH_PORT` (default `8081`)
- Common log keys: `request_id`, `trace_id`, `user_id`
