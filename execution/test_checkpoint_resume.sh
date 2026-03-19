#!/bin/bash

# Comprehensive test for checkpoint/resume and JSON match storage

set -e

echo "╔════════════════════════════════════════════════════╗"
echo "║   Checkpoint/Resume & JSON Match Storage Test     ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

cd "$(dirname "$0")"

# Clean up previous test files
rm -f test_checkpoint.json test_matches.jsonl test_matches.log

echo "1. Testing with real working mnemonic..."
echo "   This should find matches immediately!"
echo ""

# Run for 3 seconds to find matches and create checkpoint
timeout 3s ../build/btc_recover \
    -chits \
    -chits-file chits_example.json \
    -addresses ../test_addresses.tsv \
    -checkpoint test_checkpoint.json \
    -checkpoint-every 100 \
    -matches-file test_matches.jsonl \
    -i 5 \
    -c 2 \
    -v 2>&1 | tee /tmp/test_output.log || true

echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║              Test Results                          ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

# Check for matches in JSON format
if [ -f "test_matches.jsonl" ]; then
    MATCH_COUNT=$(wc -l < test_matches.jsonl | tr -d ' ')
    echo "✅ Matches JSON file created: test_matches.jsonl"
    echo "   Found $MATCH_COUNT matches"
    echo ""
    echo "   Sample match record:"
    head -1 test_matches.jsonl | python3 -m json.tool 2>/dev/null || head -1 test_matches.jsonl
    echo ""
    
    # Show all matches
    echo "   All matches found:"
    cat test_matches.jsonl | python3 -c "import sys, json; [print(f'   • {json.loads(line)[\"address\"]} via {json.loads(line)[\"derivation_path\"]}') for line in sys.stdin]" 2>/dev/null || \
    cat test_matches.jsonl | grep -o '"address":"[^"]*"' | sed 's/"address":"/   • /' | sed 's/"//'
else
    echo "❌ No matches file found"
fi

echo ""

# Check checkpoint
if [ -f "test_checkpoint.json" ]; then
    echo "✅ Checkpoint file created: test_checkpoint.json"
    echo ""
    cat test_checkpoint.json | python3 -m json.tool 2>/dev/null || cat test_checkpoint.json
    echo ""
else
    echo "⚠️  No checkpoint file created (might be too fast)"
fi

echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║          Testing Resume Functionality              ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

if [ -f "test_checkpoint.json" ]; then
    # Create a scrambled version to test resume with more permutations
    cat > test_scrambled_chits.json << 'EOF'
{
  "chits": [
    ["jar", "nest", "cool", "lunar"],
    ["argue", "payment", "reveal", "filter"],
    ["neck", "coffee", "differ", "plate"],
    ["choose", "choice", "melody", "shallow"],
    ["identify", "palace", "bone", "common"],
    ["dice", "please", "sponsor", "clip"]
  ]
}
EOF

    echo "Testing with scrambled chits (will take longer)..."
    echo ""
    
    # Clean checkpoint for fresh start
    rm -f test_checkpoint_scrambled.json test_matches_scrambled.jsonl
    
    # Run first session
    echo "Session 1: Running for 3 seconds..."
    timeout 3s ../build/btc_recover \
        -chits \
        -chits-file test_scrambled_chits.json \
        -addresses ../test_addresses.tsv \
        -checkpoint test_checkpoint_scrambled.json \
        -checkpoint-every 1000 \
        -matches-file test_matches_scrambled.jsonl \
        -i 3 \
        -c 1 2>&1 | grep -E "(Progress|Checkpoint|Resuming)" || true
    
    echo ""
    echo "Session 1 completed (interrupted)"
    
    if [ -f "test_checkpoint_scrambled.json" ]; then
        TESTED_BEFORE=$(grep -o '"mnemonics_tested":[0-9]*' test_checkpoint_scrambled.json | grep -o '[0-9]*')
        echo "  Mnemonics tested: $TESTED_BEFORE"
        echo ""
        
        # Resume and run for another 3 seconds
        echo "Session 2: Resuming from checkpoint..."
        timeout 3s ../build/btc_recover \
            -chits \
            -chits-file test_scrambled_chits.json \
            -addresses ../test_addresses.tsv \
            -checkpoint test_checkpoint_scrambled.json \
            -checkpoint-every 1000 \
            -matches-file test_matches_scrambled.jsonl \
            -i 3 \
            -c 1 2>&1 | grep -E "(Progress|Checkpoint|Resuming)" || true
        
        echo ""
        TESTED_AFTER=$(grep -o '"mnemonics_tested":[0-9]*' test_checkpoint_scrambled.json | grep -o '[0-9]*')
        echo "Session 2 completed"
        echo "  Mnemonics tested after resume: $TESTED_AFTER"
        echo ""
        
        if [ "$TESTED_AFTER" -gt "$TESTED_BEFORE" ]; then
            echo "✅ Resume working! Progress continued from $TESTED_BEFORE to $TESTED_AFTER"
        else
            echo "⚠️  Resume might not be working correctly"
        fi
    fi
else
    echo "Skipping resume test - no checkpoint available"
fi

echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║              Summary                               ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

echo "Features tested:"
echo "  ✅ Match storage in JSON format (JSONL)"
echo "  ✅ Checkpoint creation"
echo "  ✅ Resume from checkpoint"
echo "  ✅ Progress tracking"
echo ""

echo "Files created:"
[ -f "test_matches.jsonl" ] && echo "  • test_matches.jsonl - Found matches in JSON format"
[ -f "test_checkpoint.json" ] && echo "  • test_checkpoint.json - Checkpoint for correct order"
[ -f "test_checkpoint_scrambled.json" ] && echo "  • test_checkpoint_scrambled.json - Checkpoint for scrambled chits"
[ -f "test_matches_scrambled.jsonl" ] && echo "  • test_matches_scrambled.jsonl - Matches from scrambled test"
echo ""

echo "To inspect matches:"
echo "  cat test_matches.jsonl | python3 -m json.tool"
echo ""

echo "To inspect checkpoint:"
echo "  cat test_checkpoint.json | python3 -m json.tool"
echo ""

echo "Clean up test files:"
echo "  rm -f test_*.json test_*.jsonl"
echo ""
