# Skill: API Security (OWASP API Security)

## Purpose
Build secure-by-default HTTP APIs and reduce common API exploit classes.

## SSOT
- API security checklist: `docs/05_operations/API_SECURITY.md`
- Error handling: `docs/03_integration/ERROR_HANDLING.md`

## Safe defaults
- Enforce authorization in the usecase layer (ownership + business-flow constraints).
- Validate inputs at boundaries (schema/format/range) and re-check key invariants in usecases.
- Add resource limits (payload size, concurrency, timeouts) to prevent abuse.

## Common mistakes to avoid
- Gateway-only authorization (BOLA/BFLA risks).
- Returning overly detailed errors that leak internals.
- Missing inventory/versioning for public endpoints.

## Checklist
- AuthN and AuthZ responsibilities are explicit (gateway vs usecase).
- Rate limits / timeouts / size limits exist for exposed endpoints.
- Sensitive fields are not logged.
