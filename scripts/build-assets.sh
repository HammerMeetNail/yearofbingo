#!/bin/bash
# Build script for minifying CSS and JS assets
# This creates minified versions of the assets for production

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
STATIC_DIR="$PROJECT_ROOT/web/static"

# Check if we have minification tools
# We'll use a simple approach with sed/grep for basic minification
# For production, consider using tools like esbuild, terser, or csso

echo "Building assets..."

# Create dist directory
mkdir -p "$STATIC_DIR/dist/css"
mkdir -p "$STATIC_DIR/dist/js"

# Simple CSS minification (removes comments, extra whitespace)
minify_css() {
    local input="$1"
    local output="$2"

    # Remove comments, collapse whitespace, remove newlines
    cat "$input" | \
        sed 's|/\*[^*]*\*\+\([^/*][^*]*\*\+\)*/||g' | \
        tr '\n' ' ' | \
        sed 's/  */ /g' | \
        sed 's/ \?{\s*/{/g' | \
        sed 's/\s*}\s*/}/g' | \
        sed 's/;\s*/;/g' | \
        sed 's/:\s*/:/g' | \
        sed 's/,\s*/,/g' | \
        sed 's/^\s*//' | \
        sed 's/\s*$//' > "$output"

    echo "Minified: $input -> $output"
}

# Simple JS minification (removes comments, collapses some whitespace)
# Note: This is basic - for production, use a proper minifier
minify_js() {
    local input="$1"
    local output="$2"

    # Remove single-line comments (but not URLs), collapse whitespace
    cat "$input" | \
        sed 's|//[^:]*$||g' | \
        tr '\n' ' ' | \
        sed 's/  */ /g' | \
        sed 's/^\s*//' | \
        sed 's/\s*$//' > "$output"

    echo "Minified: $input -> $output"
}

# Minify CSS
for css_file in "$STATIC_DIR/css"/*.css; do
    if [ -f "$css_file" ]; then
        filename=$(basename "$css_file" .css)
        minify_css "$css_file" "$STATIC_DIR/dist/css/${filename}.min.css"
    fi
done

# Minify JS
for js_file in "$STATIC_DIR/js"/*.js; do
    if [ -f "$js_file" ]; then
        filename=$(basename "$js_file" .js)
        minify_js "$js_file" "$STATIC_DIR/dist/js/${filename}.min.js"
    fi
done

# Calculate size savings
echo ""
echo "Size comparison:"
for css_file in "$STATIC_DIR/css"/*.css; do
    if [ -f "$css_file" ]; then
        filename=$(basename "$css_file" .css)
        original_size=$(wc -c < "$css_file")
        minified_size=$(wc -c < "$STATIC_DIR/dist/css/${filename}.min.css")
        savings=$((100 - (minified_size * 100 / original_size)))
        echo "  ${filename}.css: ${original_size}B -> ${minified_size}B (${savings}% reduction)"
    fi
done

for js_file in "$STATIC_DIR/js"/*.js; do
    if [ -f "$js_file" ]; then
        filename=$(basename "$js_file" .js)
        original_size=$(wc -c < "$js_file")
        minified_size=$(wc -c < "$STATIC_DIR/dist/js/${filename}.min.js")
        savings=$((100 - (minified_size * 100 / original_size)))
        echo "  ${filename}.js: ${original_size}B -> ${minified_size}B (${savings}% reduction)"
    fi
done

echo ""
echo "Build complete! Minified assets are in $STATIC_DIR/dist/"
echo ""
echo "Note: For production deployments, consider using proper minification tools like:"
echo "  - esbuild (fast, modern bundler)"
echo "  - terser (JavaScript)"
echo "  - csso or clean-css (CSS)"
