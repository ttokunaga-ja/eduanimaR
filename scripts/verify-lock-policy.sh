#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

# Ban broad lockfile ignore patterns that break reproducibility.
if grep -nE '(^|/|\s)\*\.lock($|\s)' .gitignore >/dev/null; then
  echo "❌ .gitignore must not ignore *.lock patterns"
  exit 1
fi

# Node lockfile must exist for deterministic web installs.
if [[ ! -f apps/web/package-lock.json ]]; then
  echo "❌ apps/web/package-lock.json is required"
  exit 1
fi

# Go lock files are required and must be tracked together.
if [[ ! -f apps/professor/go.mod || ! -f apps/professor/go.sum ]]; then
  echo "❌ apps/professor/go.mod and apps/professor/go.sum are required"
  exit 1
fi

# Python lock policy is documented before lockfile adoption.
if [[ ! -f docs/DEPENDENCY_LOCK_POLICY.md ]]; then
  echo "❌ docs/DEPENDENCY_LOCK_POLICY.md is required"
  exit 1
fi

echo "✅ lock policy checks passed"
