# Refactoring Phase 1 Backlog (Pre-release Safe Track)

## Goal
Phase 1 focuses on reducing technical debt without risky, large-scale moves.

## Principles
- Keep services running while refactoring.
- Prefer additive changes over destructive moves.
- Merge in small PRs with explicit rollback points.
- Preserve contracts in `contracts/` as SSOT.

## Workstream A: Generated Code Boundaries
1. Define generated-code policy per service.
   - `apps/professor/gen/proto/` (Go)
   - `apps/librarian/src/librarian/v1/` (Python, transitional)
   - `apps/web/src/shared/api/generated/` (TypeScript)
2. Add README notes in each service about generation commands and ownership.
3. Add CI check to ensure codegen commands run successfully.

## Workstream B: Test Topology Convergence
1. Introduce `tests/unit`, `tests/integration`, `tests/contract` layout per service.
2. Migrate tests incrementally (no big-bang move).
3. Keep Make targets backward-compatible during migration.

## Workstream C: Operations Surface Consolidation
1. Create `ops/` index doc with links to compose/cloud/deploy assets.
2. Gradually move operational docs into `ops/docs/` using stubs/redirect docs.
3. Avoid moving runtime-critical files until all references are updated.

## PR Sequence (Recommended)
1. PR-1: CI/Make parity (completed baseline).
2. PR-2: Generated-code policy + docs updates.
3. PR-3: Test directory convergence scaffolding.
4. PR-4: Ops doc index and migration stubs.

## Exit Criteria
- `make verify` passes locally and in CI.
- No contract drift for OpenAPI/Proto.
- Each service has clear test topology and codegen ownership.
- New contributors can follow one documented path from setup to verification.
