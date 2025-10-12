#!/bin/bash
set -euo pipefail

# Create necessary directories if they don't exist
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "${BASE_DIR}/configs/sub"
mkdir -p "${BASE_DIR}/../sdk/core"

# Check for optional exclude-tags parameter
EXCLUDE_TAGS="compute,CDN and DNS,Sotoon Kubernetes Engine"

if [ $# -eq 1 ]; then
  EXCLUDE_TAGS="$1"
  echo "Will exclude the following tags during generation: $EXCLUDE_TAGS"
fi

# Download the OpenAPI specification from the remote URL
echo "Downloading OpenAPI specification from https://api.sotoon.ir/openapi..."
curl -s https://api.sotoon.ir/openapi -o ../configs/openapi.json

if [ $? -eq 0 ]; then
  echo "✓ Successfully downloaded OpenAPI specification"
else
  echo "✗ Failed to download OpenAPI specification"
  exit 1
fi

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
