# Benchmarking Guide

This guide explains how to benchmark the BTC Recover chits recovery feature.

## Quick Start

```bash
# Run the benchmark script
./benchmark_chits.sh
```

This will:
1. Build the latest version
2. Create test data (12-word mnemonic from 3 chits)
3. Run the recovery process
4. Test checkpoint/resume functionality
5. Display performance metrics

## What Gets Tested

### Test Configuration
- **Chits:** 3 groups of 4 words each (12-word mnemonic)
- **Search Space:** 82,944 total permutations
- **Expected Valid:** ~5,184 mnemonics (after BIP39 checksum)
- **Addresses:** 4 test addresses to search for
- **Checkpoint:** Every 10,000 tested mnemonics

### Performance Metrics

The benchmark measures:
- **Throughput:** Mnemonics tested per second
- **Validation Rate:** Percentage passing BIP39 checksum
- **Address Derivation:** Speed of BIP32 derivation
- **Lookup Performance:** Hash set query speed

### Expected Results

On modern hardware (example: AMD Ryzen 9 9950X3D):
- **CPU Mode:** 50,000-100,000 mnemonics/sec
- **Complete Test:** 1-2 seconds for full 12-word search space

## Manual Benchmarking

### Small Search Space (12-word)

```bash
./build/btc_recover \
    -chits \
    -chits-file chits_example.json \
    -addresses testdata/addresses.tsv \
    -i 20 \
    -c 5 \
    -v
```

### Large Search Space (24-word)

Create a 24-word chits file:

```json
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
```

Run with checkpoint for long-running tests:

```bash
./build/btc_recover \
    -chits \
    -chits-file my_24word_chits.json \
    -addresses addresses.tsv \
    -i 20 \
    -c 10 \
    -checkpoint my_checkpoint.json \
    -checkpoint-every 100000
```

## Testing Checkpoint/Resume

### 1. Start the benchmark

```bash
./build/btc_recover \
    -chits \
    -chits-file benchmark/benchmark_chits.json \
    -addresses benchmark/benchmark_addresses.tsv \
    -checkpoint test_checkpoint.json \
    -checkpoint-every 5000 \
    -c 2 \
    -v
```

### 2. Stop it (Ctrl+C) after a few checkpoints

```
Progress: 15234 total | 94 valid | 7617/sec
Checkpoint saved: chit perm 2, 15234 mnemonics tested
^C
```

### 3. Resume from checkpoint

Run the exact same command again. It should resume:

```
Resuming from checkpoint: chit perm 2, 15234 mnemonics tested
Progress: 25678 total | 159 valid | 10444/sec
```

## Interpreting Results

### Throughput Analysis

| Throughput | Rating | Notes |
|------------|--------|-------|
| < 10K/sec | Poor | Check CPU usage, may be I/O bound |
| 10K-50K/sec | Good | Normal for CPU-only mode |
| 50K-100K/sec | Excellent | Optimal performance |
| > 100K/sec | Outstanding | High-end CPU with good optimization |

### Checkpoint Overhead

Checkpointing has minimal impact:
- **Checkpoint write:** < 1ms
- **Frequency:** Every N mnemonics (configurable)
- **Overhead:** < 0.01% with default settings

### Memory Usage

Expected memory consumption:
- **Address HashSet (50M addresses):** ~2.5 GB
- **Worker Overhead:** ~100 MB
- **Total:** ~2.6 GB for full address list

## Comparing Modes

### Random Mode vs Chits Mode

| Metric | Random Mode | Chits Mode |
|--------|-------------|------------|
| Throughput | ~160K addresses/sec | ~50-100K mnemonics/sec |
| Per Mnemonic | 80 addresses | 80 addresses |
| Success Rate | ~0% (infinite keyspace) | High (finite keyspace) |
| Use Case | Educational demo | Actual recovery |

### CPU vs GPU (Future)

Currently only CPU mode is implemented for chits. GPU acceleration would help with:
- PBKDF2 computation (~50% of time)
- BIP32 derivation in parallel
- Batch address generation

Estimated GPU improvement: 5-10x throughput

## Troubleshooting

### Low Throughput

**Symptoms:** < 10K mnemonics/sec

**Possible Causes:**
1. High CPU usage from other processes
2. Thermal throttling
3. Too many address indexes (`-i` too high)
4. I/O contention (address file on slow disk)

**Solutions:**
```bash
# Reduce address indexes
-i 10

# Ensure address cache is loaded
# (first run will be slower while building cache)

# Monitor CPU usage
top -p $(pgrep btc_recover)
```

### Checkpoint Not Saving

**Symptoms:** No checkpoint file created

**Possible Causes:**
1. Not enough mnemonics tested to trigger checkpoint
2. Permission issues
3. Invalid checkpoint path

**Solutions:**
```bash
# Lower checkpoint frequency
-checkpoint-every 1000

# Check permissions
ls -la *.json

# Use absolute path
-checkpoint /full/path/to/checkpoint.json
```

### Resume Not Working

**Symptoms:** Starts from beginning despite checkpoint

**Possible Causes:**
1. Checkpoint file corrupted
2. Different chits file
3. Checkpoint from different run

**Solutions:**
```bash
# Verify checkpoint is valid JSON
cat checkpoint.json | python -m json.tool

# Ensure same chits file is used
# Checkpoint includes state specific to that chits configuration
```

## Performance Tips

### 1. Optimize Address Indexes

If you know your addresses are in early indexes:
```bash
-i 5  # Only check first 5 indexes (20 addresses instead of 80)
```

### 2. Reduce Derivation Paths

Edit code to only check specific paths if you know the wallet type:
- BIP44 only: Legacy wallets
- BIP84 only: Native SegWit
- BIP86 only: Taproot

### 3. Pre-load Address Cache

On first run, the address TSV is loaded into memory:
```bash
# First run (slow - builds cache)
./build/btc_recover -addresses addresses.tsv ...

# Subsequent runs (fast - uses cached data)
# Cache is in memory for duration of process
```

### 4. Use SSD for Address File

Keep the address TSV file on an SSD for faster initial load.

## Automated Testing

Create a test script:

```bash
#!/bin/bash
# test_performance.sh

echo "Testing different configurations..."

for indexes in 5 10 20; do
    echo "Testing with $indexes indexes..."
    ./build/btc_recover \
        -chits -chits-file test.json \
        -addresses test_addresses.tsv \
        -i $indexes -c 5 | grep "Progress" | tail -1
done
```

## Continuous Benchmarking

To track performance over time:

```bash
# Save results to file
./benchmark_chits.sh | tee results_$(date +%Y%m%d_%H%M%S).log

# Compare with previous runs
grep "Throughput:" results_*.log
```

## Benchmark Data

Create reproducible test chits:

```bash
# Generate test chits with known solution
# (implementation left as exercise - would require reverse-engineering)
```

## See Also

- [CHITS_MODE.md](CHITS_MODE.md) - Full chits mode documentation
- [README.md](README.md) - Main project documentation
