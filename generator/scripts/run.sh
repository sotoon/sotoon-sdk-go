#!/bin/bash
set -euo pipefail

# Create necessary directories if they don't exist
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "${BASE_DIR}/configs/sub"
mkdir -p "${BASE_DIR}/../sdk/core"

# Load exclude tags from configuration file
EXCLUDE_TAGS_FILE="${BASE_DIR}/configs/exclude-tags.json"
EXCLUDE_TAGS=""

if [ -f "$EXCLUDE_TAGS_FILE" ]; then
  # Read the exclude array from JSON and convert to comma-separated string
  EXCLUDE_TAGS=$(jq -r '.exclude | join(",")' "$EXCLUDE_TAGS_FILE")
  if [ -n "$EXCLUDE_TAGS" ] && [ "$EXCLUDE_TAGS" != "null" ]; then
    echo "Loaded exclude tags from config: $EXCLUDE_TAGS"
  fi
fi

# Allow command-line override
if [ $# -eq 1 ]; then
  EXCLUDE_TAGS="$1"
  echo "Using command-line exclude tags: $EXCLUDE_TAGS"
fi

# Download the OpenAPI specification from the remote URL
echo "Downloading OpenAPI specification from https://api.sotoon.io/openapi..."
curl -s https://api.sotoon.io/openapi -o ../configs/openapi.json

if [ $? -eq 0 ]; then
  echo "✓ Successfully downloaded OpenAPI specification"
else
  echo "✗ Failed to download OpenAPI specification"
  exit 1
fi

# Clean up old sub-API files to avoid processing stale data
echo "Cleaning up old sub-API files..."
rm -f "${BASE_DIR}/configs/sub"/*.json

# Generate sub-APIs and SDK
echo "Generating sub-APIs..."
if [ -z "$EXCLUDE_TAGS" ]; then
  ./create-subapis.sh ../configs/openapi.json ../configs/sub
else
  ./create-subapis.sh ../configs/openapi.json ../configs/sub "$EXCLUDE_TAGS"
fi

echo "Generating SDK..."
./create-sdk.sh ../configs/sub ../../sdk/core

echo "Done! SDK has been generated successfully."
