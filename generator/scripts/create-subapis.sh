#!/bin/bash
set -euo pipefail

# This script extracts all tags from an OpenAPI specification and creates
# filtered sub-API files for each tag in the specified output directory.

# Check if required arguments are provided
if [ $# -lt 2 ]; then
  echo "Usage: $0 <openapi-file> <output-directory> [exclude-tags]"
  echo "Example: $0 ../configs/openapi.json ../configs/sub"
  echo "Example with excluded tags: $0 ../configs/openapi.json ../configs/sub \"compute,internal\""
  exit 1
fi

# Get absolute paths to the script directory and filtering script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FILTERING_SCRIPT="${SCRIPT_DIR}/filtering.sh"

# Check if filtering script exists
if [ ! -f "$FILTERING_SCRIPT" ]; then
  echo "Error: Filtering script not found at $FILTERING_SCRIPT"
  exit 1
fi

# Get absolute paths to input and output
OPENAPI_FILE="$(realpath "$1")"
OUTPUT_DIR="$(realpath "$2")"

# Optional: Tags to exclude (comma-separated)
EXCLUDE_TAGS="${3:-}"

# Convert exclude tags to array for easier checking
if [ -n "$EXCLUDE_TAGS" ]; then
  IFS=',' read -ra EXCLUDE_TAGS_ARRAY <<< "$EXCLUDE_TAGS"
  echo "Will exclude the following tags: $EXCLUDE_TAGS"
else
  EXCLUDE_TAGS_ARRAY=()
fi

# Check if input file exists
if [ ! -f "$OPENAPI_FILE" ]; then
  echo "Error: OpenAPI file not found at $OPENAPI_FILE"
  exit 1
fi

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Extract all unique tags from the OpenAPI spec
# This combines tags from both the top-level tags array and path operations
echo "Extracting tags from $OPENAPI_FILE..."
TAGS_JSON=$(jq -r '(.tags[]?.name // empty), (.paths | to_entries | .[].value | to_entries | .[].value.tags[]? // empty)' "$OPENAPI_FILE" | sort -u | jq -R -s 'split("\n") | map(select(length > 0)) | unique' | jq -c '.[]')

# Check if any tags were found
if [ -z "$TAGS_JSON" ]; then
  echo "No tags found in the OpenAPI specification."
  exit 1
fi

# Process each tag
echo "Found tags:"
echo "$TAGS_JSON" | while read -r TAG_JSON; do
  # Remove quotes from JSON string
  TAG=$(echo "$TAG_JSON" | sed 's/^"\(.*\)"$/\1/')
  echo "- $TAG"
done
echo "Generating sub-API files..."

echo "$TAGS_JSON" | while read -r TAG_JSON; do
  # Remove quotes from JSON string
  TAG=$(echo "$TAG_JSON" | sed 's/^"\(.*\)"$/\1/')
  
  # Create a filename from the tag name
  # Replace spaces with hyphens and convert to lowercase for better filenames
  FILENAME=$(echo "$TAG" | tr '[:upper:]' '[:lower:]' | tr ' ' '-')
  
  # Check if this tag should be excluded
  SKIP_TAG=false
  for EXCLUDE_TAG in "${EXCLUDE_TAGS_ARRAY[@]}"; do
    # Trim whitespace from exclude tag
    EXCLUDE_TAG=$(echo "$EXCLUDE_TAG" | xargs)
    # Compare case-insensitively
    TAG_LOWER="$(echo "$TAG" | tr '[:upper:]' '[:lower:]')"
    EXCLUDE_LOWER="$(echo "$EXCLUDE_TAG" | tr '[:upper:]' '[:lower:]')"
    if [ "$TAG_LOWER" = "$EXCLUDE_LOWER" ]; then
      SKIP_TAG=true
      break
    fi
  done
  
  if [ "$SKIP_TAG" = true ]; then
    echo "Skipping excluded tag: $TAG"
    continue
  fi
  
  OUTPUT_FILE="$OUTPUT_DIR/$FILENAME.json"
  
  echo "Processing tag: $TAG -> $OUTPUT_FILE"
  
  # Call the filtering script to generate the sub-API
  "$FILTERING_SCRIPT" "$OPENAPI_FILE" "$TAG" "$OUTPUT_FILE"
  
  # Verify the output file was created
  if [ -f "$OUTPUT_FILE" ]; then
    echo "  ✓ Created $OUTPUT_FILE"
  else
    echo "  ✗ Failed to create $OUTPUT_FILE"
  fi
done

echo "Done! Sub-API files have been generated in $OUTPUT_DIR"
