# Derivation Paths Configuration

## Overview

The chits recovery mode now supports **configurable derivation paths**, allowing you to specify which Bitcoin address types to check. This can significantly speed up recovery if you know which wallet type was used.

## CLI Flag

```bash
-derivation-paths "44,49,84,86"
```

**Default:** `"44,49,84,86"` (all standard Bitcoin address types)

## Supported BIP Standards

| BIP | Purpose | Address Type | Starts With | Example |
|-----|---------|--------------|-------------|---------|
| **44** | Legacy | P2PKH | `1...` | 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa |
| **49** | Wrapped SegWit | P2SH-P2WPKH | `3...` | 3J98t1WpEZ73CNmYviecrnyiWrnqRhWNLy |
| **84** | Native SegWit | P2WPKH | `bc1q...` | bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4 |
| **86** | Taproot | P2TR | `bc1p...` | bc1p5cyxnuxmeuwuvkwfem96lqzszd02n6xdcjrs20cac6yqjjwudpxqkedrcr |

## Usage Examples

### Check All Address Types (Default)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -i 20
```

**Result:** Checks 80 addresses per mnemonic (20 indexes × 4 types)

### Check Only Native SegWit (BIP84)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "84" \
    -i 20
```

**Result:** Checks 20 addresses per mnemonic (20 indexes × 1 type)  
**Speed:** 4x faster than default!

### Check Legacy and Wrapped SegWit

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "44,49" \
    -i 20
```

**Result:** Checks 40 addresses per mnemonic (20 indexes × 2 types)  
**Speed:** 2x faster than default!

### Check Only P2SH-SegWit (Most Common)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "49" \
    -i 20
```

**Result:** Checks 20 addresses per mnemonic  
**Use when:** Your address starts with `3...`

## Address Index Configuration

The `-i` flag controls how many sequential addresses to check for each derivation path.

### Examples

#### Check First 5 Addresses

```bash
-i 5
```

**Checks:** Indexes 0, 1, 2, 3, 4 for each path  
**Total:** 5 × (number of paths)

#### Check First 20 Addresses (Default)

```bash
-i 20
```

**Checks:** Indexes 0-19 for each path  
**Total:** 20 × (number of paths)

#### Check First 100 Addresses

```bash
-i 100
```

**Checks:** Indexes 0-99 for each path  
**Total:** 100 × (number of paths)  
**Note:** Slower but more thorough

## Performance Impact

### Total Addresses Checked

```
Total = (Number of Paths) × (Address Indexes)
```

**Examples:**

| Paths | Indexes | Total | Speed Multiplier |
|-------|---------|-------|------------------|
| 4 (default) | 20 | 80 | 1x (baseline) |
| 1 | 20 | 20 | 4x faster |
| 2 | 20 | 40 | 2x faster |
| 4 | 5 | 20 | 4x faster |
| 4 | 100 | 400 | 0.2x (5x slower) |

### Optimization Strategy

1. **Know Your Wallet Type?**
   - Use only that specific BIP number
   - Example: Electrum uses BIP84 → `-derivation-paths "84"`

2. **Know Address Range?**
   - Reduce `-i` value
   - Example: Only used first 5 addresses → `-i 5`

3. **Unsure?**
   - Use defaults (all paths, 20 indexes)
   - Better to be thorough than miss the address

## Real-World Examples

### Electrum Wallet (Native SegWit)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "84" \
    -i 20
```

### Ledger/Trezor (Usually BIP49 or BIP84)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "49,84" \
    -i 20
```

### Old Bitcoin Core (Legacy)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "44" \
    -i 20
```

### Modern Multi-Sig (Usually P2SH)

```bash
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -derivation-paths "49" \
    -i 20
```

## Output Information

When you run with custom derivation paths, you'll see:

```
2026/01/18 22:08:18 Derivation paths: BIP49 (P2SH-SegWit), BIP84 (Native SegWit)
2026/01/18 22:08:18 Address indexes: 0-4
2026/01/18 22:08:18 Total addresses per mnemonic: 10
```

This confirms:
- Which paths are being checked
- The index range
- Total addresses generated per mnemonic

## Match Output

Matches now include the derivation path:

```json
{
  "timestamp": "2026-01-18T22:08:19+05:00",
  "address": "31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE",
  "address_type": "p2sh-p2wpkh/0",
  "derivation_path": "m/49'/0'/0'/0/0",
  "mnemonic": "argue payment reveal filter...",
  "private_key": "L3kzbNydMWiKsUTwKVZGRFW86bZPvGGPtbSTwfdRVSfAi4V5ZHSd",
  "public_key": "031240259f8dab8adbc96cb24a47258900cb417876b99dec48b72420c54ca5def3"
}
```

The `derivation_path` field shows exactly which path was used.

## Determining Your Wallet Type

### By Address Prefix

| Prefix | BIP | Type |
|--------|-----|------|
| `1...` | 44 | Legacy |
| `3...` | 49 | Wrapped SegWit |
| `bc1q...` | 84 | Native SegWit |
| `bc1p...` | 86 | Taproot |

### By Wallet Software

| Wallet | Default BIP | Recommendation |
|--------|-------------|----------------|
| Electrum (old) | 44 | `-derivation-paths "44"` |
| Electrum (new) | 84 | `-derivation-paths "84"` |
| Ledger | 49, 84 | `-derivation-paths "49,84"` |
| Trezor | 49, 84 | `-derivation-paths "49,84"` |
| Bitcoin Core | 44 | `-derivation-paths "44"` |
| Wasabi | 84 | `-derivation-paths "84"` |
| Samourai | 84 | `-derivation-paths "84"` |

## Advanced: Custom BIP Numbers

You can specify any BIP purpose number:

```bash
-derivation-paths "44,45,48,49,84,86"
```

**Warning:** Non-standard BIPs (not 44/49/84/86) will show a warning but still work.

## Troubleshooting

### No Matches Found

**Try:**
1. Add more derivation paths: `-derivation-paths "44,49,84,86"`
2. Increase address indexes: `-i 50` or `-i 100`
3. Verify your address is in the TSV file

### Very Slow Performance

**Try:**
1. Reduce paths: `-derivation-paths "49"` (if you know the type)
2. Reduce indexes: `-i 5` (if you know the range)
3. Both: `-derivation-paths "49" -i 5`

### "Invalid derivation paths" Error

**Check:**
- Format: Comma-separated numbers only
- Example: `"44,49,84"` ✓
- Wrong: `"BIP44,BIP49"` ✗
- Wrong: `"m/44'/0'/0'/0"` ✗

## Complete Example

```bash
# Full recovery with optimized settings
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

**This will:**
- Check BIP49 and BIP84 only (2x faster)
- Check indexes 0-19 (20 addresses per path)
- Total: 40 addresses per mnemonic
- Save progress every 50K mnemonics
- Report progress every 5 seconds
- Verbose output

## See Also

- [CHITS_MODE.md](bloat/CHITS_MODE.md) - Main chits documentation
- [CHECKPOINT_RESUME.md](bloat/CHECKPOINT_RESUME.md) - Checkpoint guide
- [QUICK_START.md](bloat/QUICK_START.md) - Getting started
- [execution/README.md](execution/README.md) - Test scripts
