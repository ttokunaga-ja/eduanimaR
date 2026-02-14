# Skill: Stack & SSOT

## Purpose
Prevent hallucinations and architecture drift by making “where truth lives” explicit.

## Scope
Applies to all proposals and changes in this template: services, gateway, CI/CD, contracts, DB, observability.

## SSOT (Source of Truth)
- Stack and versions: `docs/02_tech_stack/STACK.md`
- Clean Architecture dependency rules: `docs/01_architecture/CLEAN_ARCHITECTURE.md`
- Microservice boundaries: `docs/01_architecture/MICROSERVICES_MAP.md`
- Contracts: `docs/03_integration/API_CONTRACT_WORKFLOW.md` and `docs/03_integration/PROTOBUF_GRPC_STANDARDS.md`
- Operations posture: `docs/05_operations/OBSERVABILITY.md` and `docs/05_operations/CI_CD.md`

## Safe defaults
- Prefer the existing stack and patterns; do not introduce new frameworks/tools unless explicitly requested.
- Prefer linking to existing SSOT docs instead of duplicating content.

## Common mistakes to avoid
- “Let’s just add an ORM” (this template is SQL + sqlc by design).
- “Let’s switch tracing/logging library” without updating operations docs.
- Suggesting breaking changes to contracts without a versioning/deprecation plan.

## Checklist
### Before you change anything
- Identify which SSOT doc governs the change.
- Confirm the stack element already exists in `docs/02_tech_stack/STACK.md`.

### After you change something
- Update the relevant SSOT doc(s) if behavior/contract/ops expectations changed.
- Ensure cross-links point to SSOT pages rather than repeating rules.
