# CODEGEN_OWNERSHIP (Professor)

## SSOT
- REST contract source: `contracts/openapi/professor.yaml`
- Proto contract source: `contracts/proto/librarian/v1/librarian.proto`

## Generated Output (Current)
- Proto Go stubs: `apps/professor/gen/proto/`
- SQL codegen (sqlc): `apps/professor/internal/adapters/postgres/sqlcgen/`

## Generate Commands
```bash
cd apps/professor
make generate
# or separately:
make proto
make sqlc
```

## Ownership Rule
- Do not hand-edit generated files under `gen/proto/` or `sqlcgen/`.
- Regenerate after contract/schema updates.
- Business logic stays in `internal/domain`, `internal/usecases`, and adapters.

## CI Rule
- Contract workflows verify code generation can run successfully.
