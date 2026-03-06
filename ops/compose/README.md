# Compose Operations Index

## Canonical Files
- `ops/compose/docker-compose.yml`: development stack.
- `ops/compose/docker-compose.prod.yml`: production overlay.
- `Makefile`: compose entry commands (`make dev`, `make prod`, `make down`, etc).

## Usage
```bash
make infra
make dev
make verify
```

## Policy
- Use `ops/compose/docker-compose.yml` and `ops/compose/docker-compose.prod.yml` as the only compose manifests.
- Keep this index as the compose operator entrypoint.
