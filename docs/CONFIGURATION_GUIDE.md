# Configuration Guide

## Complete CLI Reference

### Required Flags

| Flag | Description | Example |
|------|-------------|---------|
| `-chits` | Enable chits mode | (flag, no value) |
| `-chits-file` | Path to chits JSON file | `my_chits.json` |
| `-addresses` | Path to address TSV file | `addresses.tsv` |

### Address Configuration

| Flag | Default | Description | Example |
|------|---------|-------------|---------|
| `-i` | 20 | Address indexes to check (0 to n-1) | `-i 50` |
| `-derivation-paths` | 44,49,84,86 | BIP purpose numbers | `-derivation-paths "49,84"` |

**Impact:**
```
Total addresses = (number of paths) × (index count)

Examples:
  -i 20 -derivation-paths "44,49,84,86"  → 80 addresses
  -i 20 -derivation-paths "49"           → 20 addresses (4x faster!)
  -i 5  -derivation-paths "49,84"        → 10 addresses (8x faster!)
```

### Checkpoint & Resume

| Flag | Default | Description |
|------|---------|-------------|
| `-checkpoint` | chits_checkpoint.json | Checkpoint file path |
| `-checkpoint-every` | 100000 | Save every N mnemonics |

### Output Files

| Flag | Default | Description |
|------|---------|-------------|
| `-matches-file` | matches.jsonl | JSON Lines match output |

### Progress Reporting

| Flag | Default | Description |
|------|---------|-------------|
| `-c` | 0 | Progress interval (seconds, 0=disabled) |
| `-v` | false | Verbose output |

### Notifications (Optional)

| Flag | Default | Description |
|------|---------|-------------|
| `-pt` | | Pushover application token |
| `-pu` | | Pushover user key |

## Configuration Examples

### 1. Basic Recovery (Default Settings)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv
```

**Settings:**
- Paths: BIP44, BIP49, BIP84, BIP86 (all types)
- Indexes: 0-19 (20 addresses per path)
- Total: 80 addresses per mnemonic
- Checkpoint: Every 100K mnemonics
- Progress: No reporting

### 2. Fast Recovery (Known Wallet Type)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "49" \
    -i 10
```

**Settings:**
- Paths: BIP49 only (P2SH-SegWit)
- Indexes: 0-9 (10 addresses)
- Total: 10 addresses per mnemonic
- **Speed: 8x faster than default!**

### 3. Thorough Recovery (Extended Range)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -i 100 \
    -c 10 \
    -v
```

**Settings:**
- Paths: All (default)
- Indexes: 0-99 (100 addresses per path)
- Total: 400 addresses per mnemonic
- Progress: Every 10 seconds
- Verbose: Enabled

### 4. Production Recovery (Full Features)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "49,84" \
    -i 20 \
    -checkpoint my_checkpoint.json \
    -checkpoint-every 50000 \
    -matches-file my_matches.jsonl \
    -c 5 \
    -v
```

**Settings:**
- Paths: BIP49, BIP84 (most common)
- Indexes: 0-19
- Total: 40 addresses per mnemonic
- Checkpoint: Every 50K mnemonics
- Progress: Every 5 seconds
- Custom output files

### 5. Legacy Wallet Recovery

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "44" \
    -i 50
```

**Settings:**
- Paths: BIP44 only (Legacy P2PKH)
- Indexes: 0-49 (50 addresses)
- For old Bitcoin Core wallets

### 6. Modern Wallet (SegWit Only)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "84" \
    -i 20
```

**Settings:**
- Paths: BIP84 only (Native SegWit)
- Indexes: 0-19
- For Electrum, Wasabi, Samourai

## Performance Tuning

### Speed vs Coverage Trade-off

| Configuration | Speed | Coverage | Use When |
|---------------|-------|----------|----------|
| `-derivation-paths "49" -i 5` | Fastest | Minimal | You know exact wallet type & range |
| `-derivation-paths "49,84" -i 10` | Fast | Good | Modern wallet, limited use |
| `-derivation-paths "44,49,84,86" -i 20` | Normal | Thorough | Default, unsure of type |
| `-derivation-paths "44,49,84,86" -i 100` | Slow | Maximum | Wallet with many addresses |

### Optimization Strategy

1. **Start Specific**
   ```bash
   # If you know it's a SegWit wallet
   -derivation-paths "84" -i 20
   ```

2. **Expand If No Match**
   ```bash
   # Add more paths
   -derivation-paths "49,84" -i 20
   ```

3. **Increase Range**
   ```bash
   # Check more indexes
   -derivation-paths "49,84" -i 50
   ```

4. **Full Scan (Last Resort)**
   ```bash
   # Check everything
   -derivation-paths "44,49,84,86" -i 100
   ```

## Checkpoint Configuration

### Frequency Guidelines

| Checkpoint Every | Use Case | Trade-off |
|------------------|----------|-----------|
| 1,000 | Testing | Minimal loss, slight overhead |
| 10,000 | Fast recovery | Good balance |
| 50,000 | Normal (default) | Recommended |
| 100,000 | Long recovery | Minimal overhead |
| 1,000,000 | Very long | Risk losing progress |

### Example

```bash
# For 24-word recovery (long-running)
-checkpoint-every 100000

# For 12-word recovery (fast)
-checkpoint-every 10000

# For testing
-checkpoint-every 1000
```

## Output Configuration

### Custom File Names

```bash
# Organize by date
-checkpoint recovery_$(date +%Y%m%d)_checkpoint.json \
-matches-file recovery_$(date +%Y%m%d)_matches.jsonl

# Organize by wallet
-checkpoint wallet1_checkpoint.json \
-matches-file wallet1_matches.jsonl
```

### Multiple Recovery Sessions

```bash
# Session 1: First batch of chits
./build/btc_recover \
    -chits -chits-file batch1.json \
    -checkpoint batch1_cp.json \
    -matches-file batch1_matches.jsonl \
    -addresses addresses.tsv

# Session 2: Second batch
./build/btc_recover \
    -chits -chits-file batch2.json \
    -checkpoint batch2_cp.json \
    -matches-file batch2_matches.jsonl \
    -addresses addresses.tsv

# Combine results
cat batch1_matches.jsonl batch2_matches.jsonl > all_matches.jsonl
```

## Progress Monitoring

### Real-time Progress

```bash
# Report every 5 seconds
-c 5

# Report every 30 seconds
-c 30

# No reporting (fastest)
-c 0
```

### Verbose Output

```bash
# Enable detailed logging
-v

# Shows:
# - Derivation path details
# - Address generation info
# - Checkpoint saves
# - Error messages
```

## Common Configurations

### Electrum Wallet

```bash
./build/btc_recover \
    -chits -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "84" \
    -i 20
```

### Ledger/Trezor Hardware Wallet

```bash
./build/btc_recover \
    -chits -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "49,84" \
    -i 20
```

### Bitcoin Core (Old)

```bash
./build/btc_recover \
    -chits -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "44" \
    -i 50
```

### Unknown Wallet Type

```bash
./build/btc_recover \
    -chits -chits-file my_chits.json \
    -addresses addresses.tsv \
    -i 50 \
    -c 10 \
    -v
```

## Validation

### Check Your Configuration

Before starting a long recovery, verify your settings:

```bash
# Run for 5 seconds to see configuration
timeout 5s ./build/btc_recover \
    -chits -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "49,84" \
    -i 20 \
    -v

# Look for:
# - "Derivation paths: BIP49 (P2SH-SegWit), BIP84 (Native SegWit)"
# - "Address indexes: 0-19"
# - "Total addresses per mnemonic: 40"
```

## See Also

- [DERIVATION_PATHS.md](DERIVATION_PATHS.md) - Detailed derivation path guide
- [CHECKPOINT_RESUME.md](bloat/CHECKPOINT_RESUME.md) - Checkpoint documentation
- [CHITS_MODE.md](bloat/CHITS_MODE.md) - Chits recovery guide
- [QUICK_START.md](bloat/QUICK_START.md) - Getting started
