# Dependency Lock Policy

## Purpose
Ensure reproducible builds across local development, CI, and deployment.

## Policy by Ecosystem
- Node.js (Web):
  - Lock file: `apps/web/package-lock.json`
  - Rule: Commit and review lockfile changes in PRs.

- Go (Professor):
  - Lock files: `apps/professor/go.mod`, `apps/professor/go.sum`
  - Rule: Commit both files together when dependencies change.

- Python (Librarian):
  - Current source of dependency truth: `apps/librarian/pyproject.toml`
  - Transitional lock strategy: adopt `uv.lock` as canonical lock file.
  - During transition: keep dependency updates explicit and verified by `make verify`.

## Enforcement
- `make ci-local` and CI workflows must pass before merge.
- Contract/codegen checks are part of verification path.
