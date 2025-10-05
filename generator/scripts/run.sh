#!/bin/bash
set -euo pipefail

# Create necessary directories if they don't exist
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
mkdir -p "${BASE_DIR}/configs/sub"
mkdir -p "${BASE_DIR}/../sdk/core"

# Download the OpenAPI specification from the remote URL
echo "Downloading OpenAPI specification from http://localhost:8080/openapi..."
curl -s http://localhost:8080/openapi -o ../configs/openapi.json

if [ $? -eq 0 ]; then
  echo "✓ Successfully downloaded OpenAPI specification"
else
  echo "✗ Failed to download OpenAPI specification"
  exit 1
fi

# Generate sub-APIs and SDK
echo "Generating sub-APIs..."
./create-subapis.sh ../configs/openapi.json ../configs/sub

echo "Generating SDK..."
./create-sdk.sh ../configs/sub ../../sdk/core

echo "Done! SDK has been generated successfully."
