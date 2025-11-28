#!/bin/sh
# Build script for hashing and optionally minifying CSS and JS assets
# Generates content-hashed filenames for cache busting

set -e

# Get script directory (POSIX compatible)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
STATIC_DIR="$PROJECT_ROOT/web/static"
DIST_DIR="$STATIC_DIR/dist"

echo "Building assets with content hashing..."

# Create dist directory
mkdir -p "$DIST_DIR/css"
mkdir -p "$DIST_DIR/js"

# Clear old hashed files
rm -f "$DIST_DIR/css"/*.css
rm -f "$DIST_DIR/js"/*.js
rm -f "$DIST_DIR/manifest.json"

# Function to generate short hash of file content
generate_hash() {
    _file="$1"
    # Generate MD5 hash and take first 8 characters
    if command -v md5sum >/dev/null 2>&1; then
        md5sum "$_file" | cut -c1-8
    elif command -v md5 >/dev/null 2>&1; then
        md5 -q "$_file" | cut -c1-8
    else
        # Fallback: use cksum
        cksum "$_file" | cut -d' ' -f1 | cut -c1-8
    fi
}

# Start building manifest
echo "{" > "$DIST_DIR/manifest.json"
first_entry=true

# Process CSS files
for css_file in "$STATIC_DIR/css"/*.css; do
    if [ -f "$css_file" ]; then
        filename=$(basename "$css_file" .css)
        hash=$(generate_hash "$css_file")
        hashed_name="${filename}.${hash}.css"

        # Copy file with hashed name
        cp "$css_file" "$DIST_DIR/css/${hashed_name}"

        # Add to manifest
        if [ "$first_entry" = true ]; then
            first_entry=false
        else
            echo "," >> "$DIST_DIR/manifest.json"
        fi
        printf '  "css/%s.css": "dist/css/%s"' "$filename" "$hashed_name" >> "$DIST_DIR/manifest.json"

        echo "Hashed: css/${filename}.css -> dist/css/${hashed_name}"
    fi
done

# Process JS files (excluding tests directory)
for js_file in "$STATIC_DIR/js"/*.js; do
    if [ -f "$js_file" ]; then
        filename=$(basename "$js_file" .js)
        hash=$(generate_hash "$js_file")
        hashed_name="${filename}.${hash}.js"

        # Copy file with hashed name
        cp "$js_file" "$DIST_DIR/js/${hashed_name}"

        # Add to manifest
        if [ "$first_entry" = true ]; then
            first_entry=false
        else
            echo "," >> "$DIST_DIR/manifest.json"
        fi
        printf '  "js/%s.js": "dist/js/%s"' "$filename" "$hashed_name" >> "$DIST_DIR/manifest.json"

        echo "Hashed: js/${filename}.js -> dist/js/${hashed_name}"
    fi
done

# Close manifest JSON
echo "" >> "$DIST_DIR/manifest.json"
echo "}" >> "$DIST_DIR/manifest.json"

echo ""
echo "Build complete!"
echo "Manifest written to: $DIST_DIR/manifest.json"
cat "$DIST_DIR/manifest.json"
