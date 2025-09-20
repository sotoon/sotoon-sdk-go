#!/usr/bin/env bash
set -euo pipefail

# Usage: ./filter-openapi.sh input.json "users,orders" output.json

INPUT_FILE=$1
INCLUDE_TAGS=$2
OUTPUT_FILE=$3

# Convert tags into jq array
TAGS_JSON=$(jq -Rn --arg tags "$INCLUDE_TAGS" '
  ($tags | split(",") | map(. | ltrimstr(" ") | rtrimstr(" ")))
')

# Filter paths by tag
jq --argjson tags "$TAGS_JSON" '
  .paths |= with_entries(
    .value |= with_entries(
      select(
        (.value.tags // []) as $t
        | any($t[]; . as $tag | $tags | index($tag))
      )
    )
  )
  | .paths |= with_entries(select(.value != {}))
' "$INPUT_FILE" > "$OUTPUT_FILE"

echo "Filtered spec written to $OUTPUT_FILE"
