#!/bin/bash

# Production script for chits-based seed recovery with GPU support

set -e

echo "╔════════════════════════════════════════════════════╗"
echo "║     BTC Recover - Production Recovery Tool        ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

cd "$(dirname "$0")"
cd ..  # Go to project root

# Define production directory
PROD_DIR="execution/prod"
CHITS_FILE="$PROD_DIR/chits.json"
ADDRESSES_FILE="$PROD_DIR/addresses.tsv"
CHECKPOINT_FILE="$PROD_DIR/checkpoint.json"
MATCHES_FILE="$PROD_DIR/matches.jsonl"
MATCHES_LOG="$PROD_DIR/matches.log"
OUTPUT_LOG="$PROD_DIR/recovery_output.log"

# Create production directory if it doesn't exist
echo "1. Setting up production environment..."
if [ ! -d "$PROD_DIR" ]; then
    mkdir -p "$PROD_DIR"
    echo "✓ Created production directory: $PROD_DIR"
else
    echo "✓ Production directory exists: $PROD_DIR"
fi
echo ""

# Create sample chits file if it doesn't exist
if [ ! -f "$CHITS_FILE" ]; then
    echo "2. Creating sample chits file..."
    cat > "$CHITS_FILE" << 'EOF'
{
  "chits": [
    ["word1", "word2", "word3", "word4"],
    ["word5", "word6", "word7", "word8"],
    ["word9", "word10", "word11", "word12"],
    ["word13", "word14", "word15", "word16"],
    ["word17", "word18", "word19", "word20"],
    ["word21", "word22", "word23", "word24"]
  ]
}
EOF
    echo "✓ Sample chits file created: $CHITS_FILE"
    echo ""
    echo "╔════════════════════════════════════════════════════╗"
    echo "║              ACTION REQUIRED                       ║"
    echo "╚════════════════════════════════════════════════════╝"
    echo ""
    echo "Please edit the chits file with your actual word groups:"
    echo "  File: $CHITS_FILE"
    echo ""
    echo "Each chit should contain 4 words from your mnemonic."
    echo "For a 24-word mnemonic, you need 6 chits."
    echo "For a 12-word mnemonic, you need 3 chits."
    echo ""
    read -p "Press ENTER after you have updated the chits file..."
    echo ""
else
    echo "2. Chits file already exists: $CHITS_FILE"
    echo ""
fi

# Check if addresses.tsv exists, if not, wait for it
if [ ! -f "$ADDRESSES_FILE" ]; then
    echo "3. Waiting for addresses file..."
    echo ""
    echo "╔════════════════════════════════════════════════════╗"
    echo "║              ACTION REQUIRED                       ║"
    echo "╚════════════════════════════════════════════════════╝"
    echo ""
    echo "Please create the addresses file:"
    echo "  File: $ADDRESSES_FILE"
    echo ""
    echo "Format (tab-separated):"
    echo "  address	balance"
    echo "  1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa	50"
    echo "  1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2	100"
    echo ""
    echo "Waiting for file to be created..."
    
    # Poll for file existence
    while [ ! -f "$ADDRESSES_FILE" ]; do
        sleep 2
        echo -n "."
    done
    echo ""
    echo ""
    echo "✓ Addresses file detected: $ADDRESSES_FILE"
    
    # Validate the file has content
    if [ ! -s "$ADDRESSES_FILE" ]; then
        echo "ERROR: Addresses file is empty"
        exit 1
    fi
    
    LINE_COUNT=$(wc -l < "$ADDRESSES_FILE")
    echo "  Found $LINE_COUNT lines (including header)"
    echo ""
else
    echo "3. Addresses file already exists: $ADDRESSES_FILE"
    LINE_COUNT=$(wc -l < "$ADDRESSES_FILE")
    echo "  Found $LINE_COUNT lines (including header)"
    echo ""
fi

# Detect GPU support (before building)
echo "4. Detecting GPU support..."
USE_GPU=false
GPU_FLAG=""
BUILD_TAGS=""

if command -v nvidia-smi &> /dev/null; then
    echo "✓ NVIDIA GPU detected!"
    nvidia-smi --query-gpu=name,memory.total --format=csv,noheader | head -1
    
    # Check for CUDA
    if command -v nvcc &> /dev/null; then
        CUDA_VERSION=$(nvcc --version | grep "release" | sed 's/.*release //' | sed 's/,.*//')
        echo "✓ CUDA detected: version $CUDA_VERSION"
        USE_GPU=true
        GPU_FLAG="-gpu"
        BUILD_TAGS="-tags cuda"
        echo "  Will build with GPU support (CGo + CUDA)"
    else
        echo "⚠ CUDA not found - will use CPU mode"
        echo "  Install CUDA toolkit for GPU acceleration"
    fi
else
    echo "⚠ No NVIDIA GPU detected - using CPU mode"
fi
echo ""

# Build the binary
echo "5. Building btc_recover..."
if [ ! -d "build" ]; then
    mkdir -p build
fi

if [ "$USE_GPU" = true ]; then
    echo "  Building with CUDA support..."
    CGO_ENABLED=1 go build $BUILD_TAGS -o build/btc_recover ./cmd/btc_recover
else
    echo "  Building CPU-only version..."
    go build -o build/btc_recover ./cmd/btc_recover
fi

if [ $? -ne 0 ]; then
    echo "ERROR: Build failed"
    exit 1
fi
echo "✓ Build successful: build/btc_recover"
echo ""

# Display configuration
echo "6. Configuration Summary"
echo "╔════════════════════════════════════════════════════╗"
echo "║           Recovery Configuration                   ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""
echo "Chits file:        $CHITS_FILE"
echo "Addresses file:    $ADDRESSES_FILE"
echo "Checkpoint file:   $CHECKPOINT_FILE"
echo "Matches file:      $MATCHES_FILE"
echo "Output log:        $OUTPUT_LOG"
echo "GPU acceleration:  $([ "$USE_GPU" = true ] && echo "ENABLED (built with CUDA)" || echo "DISABLED (CPU-only)")"
echo ""

# Check if resuming from checkpoint
if [ -f "$CHECKPOINT_FILE" ]; then
    echo "╔════════════════════════════════════════════════════╗"
    echo "║         Checkpoint Found - Resume Mode             ║"
    echo "╚════════════════════════════════════════════════════╝"
    echo ""
    echo "An existing checkpoint was found. The recovery will resume"
    echo "from where it left off."
    echo ""
    
    # Show checkpoint info
    if command -v jq &> /dev/null; then
        TESTED=$(jq -r '.mnemonics_tested // 0' "$CHECKPOINT_FILE")
        VALID=$(jq -r '.valid_mnemonics // 0' "$CHECKPOINT_FILE")
        ADDRESSES=$(jq -r '.addresses_checked // 0' "$CHECKPOINT_FILE")
        echo "Previous progress:"
        echo "  Mnemonics tested:    $TESTED"
        echo "  Valid mnemonics:     $VALID"
        echo "  Addresses checked:   $ADDRESSES"
        echo ""
    fi
    
    read -p "Press ENTER to continue, or Ctrl+C to abort..."
    echo ""
fi

# Final confirmation
echo "╔════════════════════════════════════════════════════╗"
echo "║              Ready to Start Recovery               ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""
echo "The recovery process will now begin. This may take a long time"
echo "depending on your chits configuration and hardware."
echo ""
echo "• The process can be interrupted at any time (Ctrl+C)"
echo "• Progress is saved automatically to checkpoint file"
echo "• To resume, simply run this script again"
echo "• All matches will be saved to: $MATCHES_FILE"
echo ""
read -p "Press ENTER to start the recovery process..."
echo ""

# Start recovery
echo "╔════════════════════════════════════════════════════╗"
echo "║              Starting Recovery                     ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""

START_TIME=$(date +%s)
START_DATE=$(date "+%Y-%m-%d %H:%M:%S")

echo "Start time: $START_DATE"
echo ""

# Build command with appropriate flags
CMD="./build/btc_recover"
CMD="$CMD -chits"
CMD="$CMD -chits-file $CHITS_FILE"
CMD="$CMD -addresses $ADDRESSES_FILE"
CMD="$CMD -checkpoint $CHECKPOINT_FILE"
CMD="$CMD -checkpoint-every 100000"
CMD="$CMD -matches-file $MATCHES_FILE"
CMD="$CMD -derivation-paths 44,49,84,86"
CMD="$CMD -i 20"
CMD="$CMD -c 10"

if [ "$USE_GPU" = true ]; then
    CMD="$CMD $GPU_FLAG"
    CMD="$CMD -gpu-batch 1024"
fi

CMD="$CMD -v"

echo "Running: $CMD"
echo ""
echo "Press Ctrl+C to stop (progress will be saved)"
echo ""

# Run the recovery (capture exit code)
set +e
$CMD 2>&1 | tee "$OUTPUT_LOG"
EXIT_CODE=$?
set -e

END_TIME=$(date +%s)
END_DATE=$(date "+%Y-%m-%d %H:%M:%S")
ELAPSED=$((END_TIME - START_TIME))

# Calculate human-readable elapsed time
HOURS=$((ELAPSED / 3600))
MINUTES=$(((ELAPSED % 3600) / 60))
SECONDS=$((ELAPSED % 60))

echo ""
echo "╔════════════════════════════════════════════════════╗"
echo "║              Recovery Complete/Stopped             ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""
echo "Start time:      $START_DATE"
echo "End time:        $END_DATE"
echo "Elapsed time:    ${HOURS}h ${MINUTES}m ${SECONDS}s"
echo "Exit code:       $EXIT_CODE"
echo ""

# Parse results
if [ -f "$OUTPUT_LOG" ]; then
    TOTAL_TESTED=$(grep -o "Progress: [0-9]* total" "$OUTPUT_LOG" | tail -1 | grep -o "[0-9]*" || echo "0")
    VALID_FOUND=$(grep -o "[0-9]* valid" "$OUTPUT_LOG" | tail -1 | grep -o "[0-9]*" || echo "0")
    
    if [ "$TOTAL_TESTED" != "0" ] && [ -n "$TOTAL_TESTED" ] && [ "$ELAPSED" -gt 0 ]; then
        RATE=$(echo "scale=2; $TOTAL_TESTED / $ELAPSED" | bc 2>/dev/null || echo "N/A")
        
        echo "Statistics:"
        echo "  Mnemonics tested:    $TOTAL_TESTED"
        echo "  Valid mnemonics:     $VALID_FOUND"
        
        if [ "$RATE" != "N/A" ]; then
            echo "  Average rate:        $RATE mnemonics/sec"
            
            # Convert to K/sec if > 1000
            if [ "$(echo "$RATE > 1000" | bc -l 2>/dev/null)" == "1" ]; then
                RATE_K=$(echo "scale=2; $RATE / 1000" | bc)
                echo "                       ${RATE_K}K mnemonics/sec"
            fi
        fi
        echo ""
    fi
fi

# Check for matches
if [ -f "$MATCHES_FILE" ]; then
    MATCH_COUNT=$(wc -l < "$MATCHES_FILE")
    if [ "$MATCH_COUNT" -gt 0 ]; then
        echo "╔════════════════════════════════════════════════════╗"
        echo "║              MATCHES FOUND!!!                      ║"
        echo "╚════════════════════════════════════════════════════╝"
        echo ""
        echo "Total matches: $MATCH_COUNT"
        echo "Matches file:  $MATCHES_FILE"
        echo ""
        echo "First match preview:"
        if command -v jq &> /dev/null; then
            head -1 "$MATCHES_FILE" | jq '.'
        else
            head -1 "$MATCHES_FILE"
        fi
        echo ""
    fi
fi

# Display checkpoint info
if [ -f "$CHECKPOINT_FILE" ]; then
    echo "Checkpoint saved: $CHECKPOINT_FILE"
    if command -v jq &> /dev/null; then
        echo "Progress details:"
        jq '.' "$CHECKPOINT_FILE" | head -20
    fi
    echo ""
fi

echo "╔════════════════════════════════════════════════════╗"
echo "║                 Output Files                       ║"
echo "╚════════════════════════════════════════════════════╝"
echo ""
echo "All files are saved in: $PROD_DIR/"
echo ""
echo "  • recovery_output.log  - Full recovery log"
echo "  • checkpoint.json      - Resume checkpoint"
echo "  • matches.jsonl        - Found matches (JSON Lines)"
echo "  • matches.log          - Found matches (text log)"
echo ""

if [ $EXIT_CODE -eq 130 ]; then
    echo "╔════════════════════════════════════════════════════╗"
    echo "║         Recovery Interrupted (Ctrl+C)              ║"
    echo "╚════════════════════════════════════════════════════╝"
    echo ""
    echo "Progress has been saved to checkpoint file."
    echo "To resume, run this script again:"
    echo "  ./execution/recover.sh"
    echo ""
elif [ $EXIT_CODE -eq 0 ]; then
    echo "Recovery completed successfully!"
    echo ""
else
    echo "Recovery exited with code: $EXIT_CODE"
    echo "Check the output log for details: $OUTPUT_LOG"
    echo ""
fi
