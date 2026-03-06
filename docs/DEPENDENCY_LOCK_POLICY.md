# Dependency Lock Policy

## Purpose
Ensure reproducible builds across local development, CI, and deployment.

## Policy by Ecosystem
- Node.js (Web):
  - Lock file: `apps/web/package-lock.json`
  - Rule: Commit and review lockfile changes in PRs. `npm ci` is the default in CI.

- Go (Professor):
  - Lock files: `apps/professor/go.mod`, `apps/professor/go.sum`
  - Rule: Commit both files together when dependencies change.

- Python (Librarian):
  - Current source of dependency truth: `apps/librarian/pyproject.toml`
  - Lock strategy: `uv.lock` will be the canonical lock file during release hardening.
  - Until `uv.lock` adoption, dependency updates must be explicit in PR and pass `make verify`.

## Repository Rule
- Do not ignore `*.lock` in `.gitignore`.
- Lock policy checks are enforced by `scripts/verify-lock-policy.sh` and included in `make verify`.

## Enforcement
- `make ci-local` and CI workflows must pass before merge.
- Contract/codegen checks are part of verification path.
