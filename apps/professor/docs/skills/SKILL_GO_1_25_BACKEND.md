# Skill: Go 1.25 Backend Conventions

## Purpose
Keep Go code idiomatic, secure-by-default, and consistent with this template’s architecture.

## SSOT
- Go guidance: `docs/02_tech_stack/GO_1_25_GUIDE.md`
- Error handling conventions: `docs/03_integration/ERROR_HANDLING.md`
- Clean Architecture direction: `docs/01_architecture/CLEAN_ARCHITECTURE.md`
- Resiliency rules: `docs/01_architecture/RESILIENCY.md`

## Safe defaults
- Use `context.Context` everywhere; enforce deadlines at boundaries (HTTP/gRPC).
- Use structured logging (`log/slog`) and never log secrets/PII.
- Prefer typed domain/usecase errors; map to transport errors in adapters.
- Keep `cmd/<service>/main.go` for wiring only.

## Common hallucinations / outdated patterns to avoid
- Global singletons for DB/clients (prefer injected dependencies).
- Panic/recover for expected errors.
- Returning raw DB/network errors directly to API responses.
- Over-abstracting IO behind “magic” helpers that reduce observability.

## Security hygiene
- Prefer Go’s official vulnerability workflow (`govulncheck`) in CI.
- Add targeted fuzz tests for parsers/validators/mappers.

## Checklist
- Deadlines/cancellation propagate from entrypoint to repositories.
- Usecases do not import transport/DB frameworks.
- Logs are structured and request-correlated (`request_id`/`trace_id`).
