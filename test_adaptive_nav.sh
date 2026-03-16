#!/bin/bash
# Quick test script for adaptive OPDS navigation

set -e

BASE_URL="${1:-http://localhost:9900/catalog}"
LIB_ID="${2:-1}"

echo "Testing Adaptive OPDS Navigation"
echo "================================="
echo "Base URL: $BASE_URL"
echo "Library ID: $LIB_ID"
echo ""

# Test 1: Root authors feed
echo "1. Testing root authors feed..."
curl -s "$BASE_URL/opds/$LIB_ID/authors" | grep -q "Авторы" && echo "✓ Root feed OK" || echo "✗ Root feed FAILED"

# Test 2: Single letter (should show either prefixes or authors)
echo "2. Testing single letter navigation (А)..."
RESPONSE=$(curl -s "$BASE_URL/opds/$LIB_ID/authors/А")
if echo "$RESPONSE" | grep -q "subsection"; then
    echo "✓ Letter А shows navigation (drill-down mode)"
elif echo "$RESPONSE" | grep -q "author:"; then
    echo "✓ Letter А shows authors (direct mode)"
else
    echo "✗ Letter А response unclear"
fi

# Test 3: Latin letter
echo "3. Testing Latin letter navigation (A)..."
RESPONSE=$(curl -s "$BASE_URL/opds/$LIB_ID/authors/A")
if echo "$RESPONSE" | grep -q "subsection\|author:"; then
    echo "✓ Letter A works"
else
    echo "✗ Letter A FAILED"
fi

# Test 4: Multi-character prefix (if applicable)
echo "4. Testing multi-character prefix (АБ)..."
RESPONSE=$(curl -s "$BASE_URL/opds/$LIB_ID/authors/АБ")
if echo "$RESPONSE" | grep -q "subsection\|author:"; then
    echo "✓ Prefix АБ works"
else
    echo "✗ Prefix АБ FAILED"
fi

echo ""
echo "Testing complete!"
echo ""
echo "Manual testing:"
echo "  - Open OPDS feed in reader: $BASE_URL/opds/$LIB_ID"
echo "  - Navigate to Authors"
echo "  - Click on letters with many authors"
echo "  - Verify drill-down navigation appears"
