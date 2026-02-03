#!/bin/bash
# Initial GitHub push script - creates a clean squashed history
# Run this ONCE to initialize the GitHub repo with clean history
# Usage: ./scripts/github-init.sh

set -e

GITHUB_REMOTE="git@github.com:sinanm89/ditong.git"
TEMP_DIR=$(mktemp -d)
CURRENT_DIR=$(pwd)

echo "=== ditong GitHub Initial Push ==="
echo "This creates a clean, squashed history for the GitHub mirror."
echo ""

# Ensure we're in the repo root
if [ ! -f "go/go.mod" ]; then
    echo "Error: Run this from the ditong repository root"
    exit 1
fi

echo "Step 1: Creating temporary clean copy..."
cp -r . "$TEMP_DIR/ditong"
cd "$TEMP_DIR/ditong"

# Remove git history
rm -rf .git

# Initialize fresh repo
git init
git config user.email "sinanm89@gmail.com"
git config user.name "Sinan Midillili"

echo "Step 2: Creating single initial commit..."
git add -A
git commit -m "feat: initial release - multi-language lexicon toolkit

ditong is a multi-language lexicon toolkit for building cross-language
word dictionaries with comprehensive metadata tracking.

Features:
- Multi-language normalization (Turkish, German, French, Spanish, etc.)
- Pluggable ingestors (Hunspell, plain text)
- Per-word metadata tracking (sources, categories, languages)
- Synthesis builder for cross-language word unions
- Pluggable IPA transcription backends
- Dual implementation (Python and Go)

Licensed under GPL-3.0 with commercial licensing available."

echo "Step 3: Adding GitHub remote..."
git remote add origin "$GITHUB_REMOTE"

echo "Step 4: Pushing to GitHub..."
git branch -M main
git push -u origin main --force

echo ""
echo "=== Done! ==="
echo "GitHub repository initialized: https://github.com/sinanm89/ditong"
echo ""
echo "Future updates will be pushed automatically on version tags via CI."

# Cleanup
cd "$CURRENT_DIR"
rm -rf "$TEMP_DIR"
