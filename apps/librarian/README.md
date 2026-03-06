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
make proto            # generate proto stubs (gen/ + src sync)
make run              # run service
make lint             # ruff check + format check
make typecheck        # mypy
make test             # unit tests
```

## Test Topology
- `tests/unit/`
- `tests/integration/` (scaffold)
- `tests/contract/` (scaffold)
