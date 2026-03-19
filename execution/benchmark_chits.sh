#!/bin/bash

# Benchmark script for chits-based seed recovery

set -e

echo "╔════════════════════════════════════════════════════╗"
echo "║     BTC Recover Chits Benchmark Tool              ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

cd "$(dirname "$0")"
cd ..  # Go to project root

# Build the binary
echo "1. Building btc_recover..."
echo ""

go build -o build/btc_recover ./cmd/btc_recover
if [ $? -ne 0 ]; then
    echo "ERROR: Build failed"
    exit 1
fi

echo "✓ Build successful: build/btc_recover"
echo ""

# Create benchmark directory
if [ ! -d "benchmark" ]; then
  mkdir "benchmark"
fi

# Clean old benchmark files
rm -f benchmark/benchmark_*

# Create test data
echo "2. Creating test data..."

# Test chits - uses REAL mnemonic that derives to known address!
# Mnemonic: argue payment reveal filter dice please sponsor clip choose choice melody shallow identify palace bone common jar nest cool lunar neck coffee differ plate
# Derives to: 31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE at m/49'/0'/0'/0/0
cat > benchmark/benchmark_chits.json << 'EOF'
{
  "chits": [
    ["argue", "payment", "reveal", "filter"],
    ["dice", "please", "sponsor", "clip"],
    ["choose", "choice", "melody", "shallow"],
    ["identify", "palace", "bone", "common"],
    ["jar", "nest", "cool", "lunar"],
    ["neck", "coffee", "differ", "plate"]
  ]
}
EOF

# Address index with the MATCHING address from the test mnemonic
cat > benchmark/benchmark_addresses.tsv << 'EOF'
address	balance
31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE	100
1PbJ41eNd1dqVeYLaFgxDUC7BgWk3RQZ6d	0
36Z9MvdvpndEJmzZMKH6xTMVTnU7od6TpK	0
EOF

echo "✓ Test data created"
echo "  - 6 chits of 4 words (24-word mnemonic)"
echo "  - Test address that SHOULD be found at index 0"
echo "  - Address: 31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE"
echo ""

# Calculate expected search space
echo "3. Search Space Analysis..."
echo "  • Chit permutations: 6! = 720"
echo "  • Word permutations per chit: 4! = 24"
echo "  • Total word permutations: 24⁶ = 191,102,976"
echo "  • Total permutations: 720 × 191,102,976 = 137,594,142,720"
echo "  • Expected valid (after checksum): ~537,477,901"
echo "  • SHORTCUT: If chits are in correct order, should find on 1st try!"
echo ""

# Run benchmark
echo "4. Running benchmark..."
echo "   Press Ctrl+C to stop and test checkpoint/resume"
echo ""

# Run with time measurement
START_TIME=$(date +%s)

./build/btc_recover \
    -chits \
    -chits-file benchmark/benchmark_chits.json \
    -addresses benchmark/benchmark_addresses.tsv \
    -i 2 \
    -c 5 \
    -checkpoint benchmark/benchmark_checkpoint.json \
    -checkpoint-every 10000 \
    -derivation-paths "49" \
    -v 2>&1 | tee benchmark/benchmark_output.log

END_TIME=$(date +%s)

# Calculate elapsed time
ELAPSED=$((END_TIME - START_TIME))

echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║              Benchmark Results                     ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

# Parse results from log
if [ -f "benchmark/benchmark_output.log" ]; then
    # Extract statistics from log
    TOTAL_TESTED=$(grep -o "Progress: [0-9]* total" benchmark/benchmark_output.log | tail -1 | grep -o "[0-9]*" || echo "0")
    VALID_FOUND=$(grep -o "[0-9]* valid" benchmark/benchmark_output.log | tail -1 | grep -o "[0-9]*" || echo "0")
    
    if [ "$TOTAL_TESTED" != "0" ] && [ -n "$TOTAL_TESTED" ]; then
        RATE=$(echo "scale=2; $TOTAL_TESTED / $ELAPSED" | bc 2>/dev/null || echo "N/A")
        
        echo "Time elapsed:        ${ELAPSED}s"
        echo "Total tested:        $TOTAL_TESTED mnemonics"
        echo "Valid found:         $VALID_FOUND mnemonics"
        echo "Throughput:          $RATE mnemonics/second"
        echo ""
        
        # Check if we found the expected number
        EXPECTED_VALID=5184
        PROGRESS_PCT=$(echo "scale=2; $TOTAL_TESTED * 100 / 82944" | bc 2>/dev/null || echo "N/A")
        echo "Progress:            ${PROGRESS_PCT}% of search space"
        echo "Expected valid:      ~$EXPECTED_VALID mnemonics"
        
        # Convert to K/sec
        if [ "$RATE" != "N/A" ] && [ "$(echo "$RATE > 1000" | bc -l 2>/dev/null)" == "1" ]; then
            RATE_K=$(echo "scale=2; $RATE / 1000" | bc)
            echo "                     ${RATE_K}K mnemonics/second"
        fi
    else
        echo "No test data found in log (may have been stopped immediately)"
    fi
    
    # Check for checkpoint
    if [ -f "benchmark/benchmark_checkpoint.json" ]; then
        echo ""
        echo "Checkpoint saved:"
        cat benchmark/benchmark_checkpoint.json | grep -E "(tested|generated|checked)" || true
    fi
    
    # Check for matches
    if [ -f "matches.log" ]; then
        MATCH_COUNT=$(wc -l < matches.log)
        echo ""
        echo "MATCHES FOUND: $MATCH_COUNT"
        echo "Check matches.log for details"
    fi
else
    echo "Benchmark log not found"
fi

echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║           Testing Checkpoint/Resume                ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

if [ -f "benchmark/benchmark_checkpoint.json" ]; then
    echo "Testing resume functionality..."
    echo "Running again with same checkpoint file..."
    echo ""
    
    START_TIME2=$(date +%s)
    
    timeout 10s ./build/btc_recover \
        -chits \
        -chits-file benchmark/benchmark_chits.json \
        -addresses benchmark/benchmark_addresses.tsv \
        -i 20 \
        -checkpoint benchmark/benchmark_checkpoint.json \
        -checkpoint-every 10000 \
        -v 2>&1 | tee benchmark/benchmark_resume.log || true
    
    END_TIME2=$(date +%s)
    
    # Check if it resumed from checkpoint
    if grep -q "Resuming from checkpoint" benchmark/benchmark_resume.log; then
        echo ""
        echo "✓ Checkpoint/resume working correctly!"
        RESUMED_FROM=$(grep "Resuming from checkpoint" benchmark/benchmark_resume.log | grep -o "[0-9]* mnemonics" | grep -o "[0-9]*")
        echo "  Resumed from: $RESUMED_FROM mnemonics tested"
    else
        echo "⚠ Could not verify checkpoint resume"
    fi
else
    echo "No checkpoint file found - run benchmark longer to test resume"
fi

echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║              Benchmark Complete                    ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""
echo "Files created:"
echo "  • benchmark/benchmark_output.log - Full output"
echo "  • benchmark/benchmark_checkpoint.json - Resume checkpoint"
echo "  • matches.log - Any found matches (if any)"
echo ""
echo "To test resume: Kill the process and run again with same flags"
echo ""
