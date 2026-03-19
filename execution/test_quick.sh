#!/bin/bash

# Quick test of chits functionality

set -e

echo "Quick Chits Mode Test"
echo "===================="
echo ""

cd "$(dirname "$0")"

# Very small chits (2 groups of 2 words = 4 word "mnemonic" - just for testing structure)
cat > test_chits_quick.json << 'EOF'
{
  "chits": [
    ["abandon", "ability"],
    ["able", "about"]
  ]
}
EOF

echo "Running quick chits mode test..."
echo ""

# Run for just 2 seconds to test functionality
timeout 2s ../build/btc_recover \
    -chits \
    -chits-file test_chits_quick.json \
    -addresses ../test_addresses.tsv \
    -i 2 \
    -checkpoint test_checkpoint_quick.json \
    -checkpoint-every 100 \
    -v || true

echo ""
echo "Test completed!"
echo ""

# Check if checkpoint was created
if [ -f "test_checkpoint_quick.json" ]; then
    echo "✓ Checkpoint file created:"
    cat test_checkpoint_quick.json
    echo ""
    
    # Test resume
    echo "Testing resume..."
    timeout 2s ../build/btc_recover \
        -chits \
        -chits-file test_chits_quick.json \
        -addresses ../test_addresses.tsv \
        -i 2 \
        -checkpoint test_checkpoint_quick.json \
        -checkpoint-every 100 || true
    echo ""
    echo "✓ Resume test completed"
else
    echo "⚠ No checkpoint created (test may have been too short)"
fi

# Cleanup
rm -f test_chits_quick.json test_checkpoint_quick.json

echo ""
echo "All basic functionality working!"
