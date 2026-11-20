#!/bin/sh
# Run from either .codecrafters/run.sh or the generated your_program.sh.
set -e

PROJECT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
if [ "$(basename "$PROJECT_DIR")" = ".codecrafters" ]; then
  PROJECT_DIR="$(dirname "$PROJECT_DIR")"
fi

PYTHONSAFEPATH=1 PYTHONPATH="$PROJECT_DIR" exec uv run \
  --project "$PROJECT_DIR" --quiet -m app.main "$@"
