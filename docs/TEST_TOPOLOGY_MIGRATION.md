# Test Topology Migration Guide

## Purpose
Converge services to a common test topology with minimal disruption:
- `tests/unit`
- `tests/integration`
- `tests/contract`

## Current State
- Librarian: `tests/unit` exists, integration/contract scaffolds added.
- Professor: tests mainly under `internal/*` and `internal/contracttest`, scaffold added under `tests/*`.
- Web: unit tests in `src/**`, E2E in `e2e/`, contract scaffold added under `tests/contract`.

## Migration Rules
1. Do not move all tests at once.
2. Keep Make targets backward-compatible during migration.
3. Preserve CI stability; each PR must pass `make verify`.
4. Prefer small PRs by feature/usecase boundary.

## Recommended Steps
1. Add new tests to target topology first.
2. Move existing tests module-by-module.
3. Remove legacy locations only after CI parity is confirmed.

## Done Criteria
- New tests are placed in topology-compliant paths.
- Legacy locations are documented until fully removed.
- CI and local verify run the same logical checks.
