#!/bin/bash
# lexis.sh — thin wrapper around the attorney's local Lexis+ CLI (lexis-plus-pp-cli) for /LexisResearch.
#
# Usage: lexis.sh <lexis-plus-pp-cli arguments...>
#   lexis.sh doctor --agent
#   lexis.sh cite check "27 Cal.3d 167" "Cal Lab Code 1102.5" --agent
#   lexis.sh citecheck text "27 Cal.3d 167" --pages 175-176
#
# It finds the CLI, puts press-auth (which holds the Lexis+ login) on PATH, and passes every
# argument through unchanged. Exit 3 means the CLI is missing.

CLI="${LEXIS_PLUS_CLI:-}"
if [ -z "$CLI" ]; then
  for c in "$HOME/printing-press/library/lexis-plus/lexis-plus-pp-cli" "$(command -v lexis-plus-pp-cli 2>/dev/null)" "$HOME/go/bin/lexis-plus-pp-cli"; do
    if [ -n "$c" ] && [ -x "$c" ]; then CLI="$c"; break; fi
  done
fi
export PATH="$HOME/go/bin:$PATH"

if [ -z "$CLI" ] || [ ! -x "$CLI" ]; then
  echo "LEXISRESEARCH_ERROR: the Lexis+ CLI (lexis-plus-pp-cli) was not found." >&2
  echo "Install it from library/other/lexis-plus in printing-press-library, or set LEXIS_PLUS_CLI to its path." >&2
  exit 3
fi

exec "$CLI" "$@"
