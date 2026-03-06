# Skill: Contracts (OpenAPI + Orval)

## Purpose
Keep external HTTP/JSON APIs contract-first and safely evolvable.

## SSOT
- OpenAPI workflow: `docs/03_integration/API_CONTRACT_WORKFLOW.md`
- API generation: `docs/03_integration/API_GEN.md`
- Error codes: `docs/03_integration/ERROR_CODES.md`

## Safe defaults
- OpenAPI is the SSOT for external APIs.
- Prefer additive changes; deprecate before removal.
- Keep response/error shapes stable and explicitly documented.

## Common mistakes to avoid
- “Just change the response shape” without versioning/deprecation.
- Divergence between gateway implementation and `openapi.yaml`.
- Silent behavior changes without updating API lifecycle docs.

## Checklist
- Update OpenAPI first, then regenerate clients (e.g., Orval).
- Confirm error codes and status mapping remain consistent.
- Provide deprecation notes and migration path for consumers.
