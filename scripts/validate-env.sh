#!/usr/bin/env sh
set -eu

SCHEMA_FILE="${1:-.env.schema}"
ENV_FILE="${2:-.env}"

if [ ! -f "$SCHEMA_FILE" ]; then
  echo "❌ schema file not found: $SCHEMA_FILE" >&2
  exit 1
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "❌ env file not found: $ENV_FILE" >&2
  echo "   run: cp .env.example .env" >&2
  exit 1
fi

missing_required=0

while IFS='|' read -r key required default_value description; do
  case "$key" in
    ""|\#*)
      continue
      ;;
  esac

  value=$(awk -F'=' -v k="$key" '
    $0 ~ /^[[:space:]]*#/ {next}
    $1 == k {
      sub(/^[^=]*=/, "", $0)
      print $0
      found=1
      exit
    }
    END { if (!found) print "" }
  ' "$ENV_FILE")

  if [ "$required" = "required" ] && [ -z "$value" ]; then
    echo "❌ required env var is missing: $key" >&2
    missing_required=1
  fi
done < "$SCHEMA_FILE"

if [ "$missing_required" -ne 0 ]; then
  echo "❌ env validation failed" >&2
  exit 1
fi

echo "✅ env validation passed ($ENV_FILE)"
