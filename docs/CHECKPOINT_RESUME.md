# Checkpoint & Resume Guide

## Overview

The chits recovery mode includes **automatic checkpoint/resume** functionality that allows you to:
- Stop and resume recovery at any time (Ctrl+C)
- Survive crashes and system reboots
- Track progress across sessions
- Store all matches in structured JSON format

## Quick Start

### Basic Usage with Checkpointing

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -checkpoint my_checkpoint.json \
    -checkpoint-every 50000 \
    -matches-file my_matches.jsonl \
    -c 5
```

**To interrupt**: Press `Ctrl+C`  
**To resume**: Run the **exact same command** again!

## How It Works

### 1. Checkpoint File

The checkpoint file (JSON format) stores:

```json
{
  "chit_perm_index": 42,
  "word_perm_indices": null,
  "mnemonics_generated": 1250,
  "mnemonics_tested": 315420,
  "addresses_checked": 25200,
  "timestamp": "2026-01-18T21:47:24Z"
}
```

**Fields:**
- `chit_perm_index`: Which chit permutation is being processed
- `mnemonics_generated`: Valid BIP39 mnemonics found
- `mnemonics_tested`: Total permutations tested (includes invalid)
- `addresses_checked`: Total addresses derived and checked
- `timestamp`: When checkpoint was saved

### 2. Matches File (JSON Lines)

Each match is saved as one JSON object per line:

```json
{
  "timestamp": "2026-01-18T21:47:22+05:00",
  "address": "31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE",
  "address_type": "p2sh-p2wpkh/0",
  "derivation_path": "m/49'/0'/0'/0/0",
  "mnemonic": "argue payment reveal filter dice please sponsor clip...",
  "private_key": "L5eCmQSpiVxXH6E8PG4jp5xFFjeeexE3qjRBLK8...",
  "public_key": "03b1620f2be614fa2d80f9f05608f77b832c..."
}
```

**Why JSON Lines (JSONL)?**
- Each line is a complete JSON object
- Easy to append new matches
- Can process with standard tools: `jq`, `python`, etc.
- Crash-safe (partially written line can be discarded)

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-checkpoint` | `chits_checkpoint.json` | Path to checkpoint file |
| `-checkpoint-every` | `100000` | Save checkpoint every N mnemonics |
| `-matches-file` | `matches.jsonl` | Path to JSON Lines matches file |

## Common Workflows

### Workflow 1: Long-Running Recovery with Checkpoints

```bash
# Start recovery with frequent checkpoints (every 10K mnemonics)
./build/btc_recover \
    -chits \
    -chits-file my_24word_chits.json \
    -addresses addresses.tsv \
    -checkpoint recovery_checkpoint.json \
    -checkpoint-every 10000 \
    -matches-file recovery_matches.jsonl \
    -c 10 \
    -v

# ... later, after interruption ...

# Resume (same command)
./build/btc_recover \
    -chits \
    -chits-file my_24word_chits.json \
    -addresses addresses.tsv \
    -checkpoint recovery_checkpoint.json \
    -checkpoint-every 10000 \
    -matches-file recovery_matches.jsonl \
    -c 10 \
    -v
```

### Workflow 2: Multiple Recovery Sessions

```bash
# Session 1: Try first batch of chits
./build/btc_recover \
    -chits \
    -chits-file batch1_chits.json \
    -checkpoint batch1_checkpoint.json \
    -matches-file batch1_matches.jsonl \
    -addresses addresses.tsv

# Session 2: Try different chits
./build/btc_recover \
    -chits \
    -chits-file batch2_chits.json \
    -checkpoint batch2_checkpoint.json \
    -matches-file batch2_matches.jsonl \
    -addresses addresses.tsv

# Combine all matches
cat batch1_matches.jsonl batch2_matches.jsonl > all_matches.jsonl
```

### Workflow 3: Distributed Recovery

```bash
# Machine 1: Process first half of permutations
# (manually edit chits order for different starting point)

# Machine 2: Process second half
# (different chits order)

# Later: Combine results
scp machine1:matches.jsonl matches1.jsonl
scp machine2:matches.jsonl matches2.jsonl
cat matches1.jsonl matches2.jsonl > combined_matches.jsonl
```

## Processing Matches

### View All Matches

```bash
# Pretty print all matches
cat matches.jsonl | jq '.'

# List only addresses
cat matches.jsonl | jq -r '.address'

# Group by derivation path
cat matches.jsonl | jq -r '[.derivation_path, .address] | @tsv'
```

### Extract Specific Match

```bash
# Find match for specific address
cat matches.jsonl | jq 'select(.address == "31qT6Xr...")'

# Get mnemonic for first match
cat matches.jsonl | head -1 | jq -r '.mnemonic'

# Export to CSV
cat matches.jsonl | jq -r '[.timestamp, .address, .derivation_path] | @csv' > matches.csv
```

### Python Processing

```python
import json

# Read all matches
matches = []
with open('matches.jsonl', 'r') as f:
    for line in f:
        matches.append(json.loads(line))

# Print summary
print(f"Total matches: {len(matches)}")
for match in matches:
    print(f"{match['address']} via {match['derivation_path']}")
    
# Export to different format
import csv
with open('matches.csv', 'w') as f:
    writer = csv.DictWriter(f, fieldnames=['address', 'derivation_path', 'mnemonic'])
    writer.writeheader()
    writer.writerows(matches)
```

## Checkpoint Management

### Check Progress

```bash
# View checkpoint info
cat checkpoint.json | jq '.'

# See how far along
TESTED=$(cat checkpoint.json | jq '.mnemonics_tested')
TOTAL=137594142720  # Total for 24-word with 6 chits
PERCENT=$(echo "scale=4; $TESTED * 100 / $TOTAL" | bc)
echo "Progress: ${PERCENT}%"
```

### Manual Checkpoint Manipulation

```bash
# Reset to beginning (dangerous!)
rm checkpoint.json

# Save backup
cp checkpoint.json checkpoint_backup_$(date +%Y%m%d_%H%M%S).json

# Resume from specific point (advanced)
cat checkpoint.json | jq '.chit_perm_index = 100' > checkpoint_modified.json
```

## Troubleshooting

### Checkpoint Not Being Created

**Problem**: No checkpoint file after running

**Solutions**:
1. Check you're using `-checkpoint` flag
2. Lower `-checkpoint-every` value (try 1000)
3. Run longer (checkpoint saves periodically)
4. Check file permissions

```bash
# Test with frequent checkpoints
-checkpoint-every 100
```

### Resume Not Working

**Problem**: Starts from beginning despite checkpoint

**Causes**:
1. Different chits file
2. Checkpoint file corrupted
3. File path mismatch

**Solutions**:
```bash
# Verify checkpoint is valid JSON
cat checkpoint.json | python3 -m json.tool

# Check the command matches exactly
# (same -chits-file, same -checkpoint path)

# Verify checkpoint is being loaded
# Look for "Resuming from checkpoint" in output
```

### Matches File Corrupted

**Problem**: JSON parsing errors

**Solution**: Last line might be incomplete

```bash
# Remove last line if incomplete
head -n -1 matches.jsonl > matches_fixed.jsonl

# Verify each line is valid JSON
cat matches.jsonl | while read line; do
    echo "$line" | python3 -m json.tool > /dev/null || echo "Invalid: $line"
done
```

## Performance Considerations

### Checkpoint Frequency

| Frequency | Pros | Cons |
|-----------|------|------|
| Every 1K | Minimal loss on crash | Slight performance hit |
| Every 10K | Good balance | ~10 seconds lost |
| Every 100K | Best performance | ~100 seconds lost |
| Every 1M | Minimal overhead | Minutes of work lost |

**Recommendation**: 
- For 12-word (fast): 10,000
- For 24-word (slow): 50,000-100,000

### Checkpoint Overhead

Checkpoint saves are very fast (< 1ms) but happen synchronously.

```
Overhead = (checkpoint_time × saves_per_second)
         = 1ms × (mnemonics_per_sec / checkpoint_every)
         = 1ms × (100,000 / 100,000)
         = 1ms overhead per second = 0.1%
```

With default settings, overhead is **negligible** (< 0.1%).

## Security Considerations

### ⚠️ Critical: Secure Your Files

When matches are found:

```bash
# 1. IMMEDIATELY copy to secure location
cp matches.jsonl /secure/encrypted/volume/

# 2. Extract just the mnemonic
cat matches.jsonl | jq -r '.mnemonic' | head -1 > SECURE_THIS.txt

# 3. Test the recovery
# (use a wallet to verify it works)

# 4. DELETE ALL FILES
shred -u matches.jsonl matches.log checkpoint.json
# or on macOS:
rm -P matches.jsonl matches.log checkpoint.json
```

### What Gets Saved

**Sensitive data in files**:
- ✅ `matches.jsonl` - **CRITICAL** - Contains private keys!
- ✅ `matches.log` - **CRITICAL** - Contains mnemonics and private keys!
- ⚠️ `checkpoint.json` - Contains progress only (no keys)

**Safe to backup**:
- Checkpoint file (contains no sensitive data)
- Chits file (partial info only)
- Address file (public addresses)

## Advanced: Custom Checkpoint Logic

If you need more control, you can modify the checkpoint behavior in the code:

```go
// Change checkpoint frequency dynamically
checkpointEvery := 100000
if matchesFound > 0 {
    checkpointEvery = 1000  // More frequent after finding matches
}
```

## Comparison with Other Tools

| Feature | btc_recover | Other Recovery Tools |
|---------|-------------|---------------------|
| Checkpoint Format | JSON (human-readable) | Binary/custom |
| Match Format | JSON Lines | CSV/Text |
| Auto-resume | ✅ Yes | ⚠️ Manual |
| Progress Tracking | ✅ Detailed | ⚠️ Basic |
| Crash Recovery | ✅ Automatic | ❌ No |

## See Also

- [CHITS_MODE.md](CHITS_MODE.md) - Main chits documentation
- [QUICK_START.md](QUICK_START.md) - Getting started guide
- [BENCHMARK.md](BENCHMARK.md) - Performance tuning
- [TEST_VALIDATION.md](TEST_VALIDATION.md) - Test cases
