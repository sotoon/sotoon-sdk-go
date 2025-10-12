#!/bin/bash
set -euo pipefail

# This script generates Go SDK code using oapi-codegen for each sub-API specification
# in the specified directory. It generates both client and types code.

# Check if required arguments are provided
if [ $# -lt 2 ]; then
  echo "Usage: $0 <sub-api-directory> <output-directory>"
  echo "Example: $0 ../configs/sub ../sdk"
  exit 1
fi

SUB_API_DIR="$(realpath "$1")"
OUTPUT_DIR="$(realpath "$2")"

if [ ! -d "$SUB_API_DIR" ]; then
  echo "Error: Sub-API directory not found at $SUB_API_DIR"
  exit 1
fi

mkdir -p "$OUTPUT_DIR"

# Get the script directory to find the name mapping file
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAME_MAPPING_FILE="$SCRIPT_DIR/../configs/name-mapping.json"

# Check if name mapping file exists
if [ -f "$NAME_MAPPING_FILE" ]; then
  echo "Using name mapping from $NAME_MAPPING_FILE"
else
  echo "Warning: Name mapping file not found at $NAME_MAPPING_FILE"
  echo "Proceeding without name mapping..."
fi

# Check if oapi-codegen is installed
if ! command -v oapi-codegen &> /dev/null; then
  echo "Error: oapi-codegen is not installed. Please install it using:"
  echo "go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest"
  exit 1
fi

# Find all JSON files in the sub-API directory
SUB_API_FILES=$(find "$SUB_API_DIR" -name "*.json")

# Check if any sub-API files were found
if [ -z "$SUB_API_FILES" ]; then
  echo "No sub-API files found in $SUB_API_DIR"
  exit 1
fi

echo "Found $(echo "$SUB_API_FILES" | wc -l | tr -d ' ') sub-API files"
echo "Generating SDK code..."

# Process each sub-API file
for API_FILE in $SUB_API_FILES; do
  # Get the filename without extension
  FILENAME=$(basename "$API_FILE" .json)
  
  # Check if there's a name mapping for this file
  MAPPED_NAME="$FILENAME"
  if [ -f "$NAME_MAPPING_FILE" ]; then
    # Try to get the mapped name from the JSON file
    MAPPED_NAME=$(jq -r --arg key "$FILENAME" '.[$key] // $key' "$NAME_MAPPING_FILE")
  fi
  
  # Create a package name from the mapped name (ensure it's a valid Go package name)
  PACKAGE_NAME=$(echo "$MAPPED_NAME" | tr '-' '_')
  
  # Create a directory for this API's SDK using the mapped name
  API_OUTPUT_DIR="$OUTPUT_DIR/$MAPPED_NAME"
  mkdir -p "$API_OUTPUT_DIR"
  
  if [ "$FILENAME" != "$MAPPED_NAME" ]; then
    echo "Processing $FILENAME -> $MAPPED_NAME..."
  else
    echo "Processing $FILENAME..."
  fi
  
  # Generate client code
  echo "  Generating client code..."
  CLIENT_FILE="$API_OUTPUT_DIR/client.gen.go"
  if oapi-codegen -generate client -package "$PACKAGE_NAME" "$API_FILE" > "$CLIENT_FILE"; then
    echo "  ✓ Generated client code: $CLIENT_FILE"
  else
    echo "  ✗ Failed to generate client code for $FILENAME"
    continue
  fi
  
  # Generate types code
  echo "  Generating types code..."
  TYPES_FILE="$API_OUTPUT_DIR/types.gen.go"
  if oapi-codegen -generate types -package "$PACKAGE_NAME" "$API_FILE" > "$TYPES_FILE"; then
    echo "  ✓ Generated types code: $TYPES_FILE"
  else
    echo "  ✗ Failed to generate types code for $FILENAME"
  fi
  
  # Generate handler code
  echo "  Generating handler code..."
  HANDLER_FILE="$API_OUTPUT_DIR/handler.go"
  if go run generate-handler.go "$PACKAGE_NAME" "$HANDLER_FILE"; then
    echo "  ✓ Generated handler code: $HANDLER_FILE"
  else
    echo "  ✗ Failed to generate handler code for $FILENAME"
  fi
done

# Generate main SDK wrapper file
echo "Generating main SDK wrapper..."
SDK_WRAPPER_FILE="$OUTPUT_DIR/../sdk.go"
if go run generate-sdk.go "$OUTPUT_DIR" "$SDK_WRAPPER_FILE"; then
  echo "✓ Generated SDK wrapper: $SDK_WRAPPER_FILE"
else
  echo "✗ Failed to generate SDK wrapper"
fi

echo "Done! SDK code has been generated in $OUTPUT_DIR"
