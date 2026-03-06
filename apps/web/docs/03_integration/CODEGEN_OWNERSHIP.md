# CODEGEN_OWNERSHIP (Web)

## SSOT
- OpenAPI contract source: `contracts/openapi/professor.yaml`

## Generated Output (Current)
- API client and schemas: `apps/web/src/shared/api/generated/`

## Generate Command
```bash
cd apps/web
npm run api:generate
```

## Ownership Rule
- Do not hand-edit generated files.
- Update contracts first, then regenerate clients.
- Hand-written API wrappers live outside `src/shared/api/generated/`.

## CI Rule
- Contract workflows validate OpenAPI codegen execution.
