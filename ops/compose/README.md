# Compose Operations Index

## Canonical Files
- `docker-compose.yml`: development stack.
- `docker-compose.prod.yml`: production overlay.
- `Makefile`: compose entry commands (`make dev`, `make prod`, `make down`, etc).

## Usage
```bash
make infra
make dev
make verify
```

## Migration Plan
- Keep compose files at repository root until all references are updated.
- Use this index as stable operator entrypoint.
