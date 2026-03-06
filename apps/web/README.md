# eduanima-web

Next.js (TypeScript/FSD) frontend for eduanimaR.

## Responsibility
- User-facing UI for subjects, materials, and chat workflows.
- Uses generated API clients from OpenAPI contract.

## Contracts
- OpenAPI SSOT: `../../contracts/openapi/professor.yaml`
- Service docs: `docs/README.md`

## Common Commands
```bash
npm run dev           # start dev server
npm run api:generate  # generate API client (gen/openapi/web)
npm run lint          # eslint
npm run typecheck     # tsc --noEmit
npm run test          # vitest
npm run build         # production build
```

## Test Topology
- Unit/component tests: `src/**/*.test.ts(x)`
- E2E tests: `e2e/`
- Contract tests: `tests/contract/` (scaffold)

## Health And Observability
- Health endpoints: `GET /api/healthz`, `GET /api/readyz`
- Response keys include: `request_id`, `trace_id`, `user_id`
