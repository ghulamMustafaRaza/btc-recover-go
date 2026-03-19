# Quick Start: Chits Recovery Mode

## TL;DR

```bash
# 1. Create your chits file
cat > my_chits.json << EOF
{
  "chits": [
    ["word1", "word2", "word3", "word4"],
    ["word5", "word6", "word7", "word8"],
    ["word9", "word10", "word11", "word12"]
  ]
}
EOF

# 2. Download addresses (if you don't have it)
curl -L -o addresses.tsv.gz \
  http://addresses.loyce.club/blockchair_bitcoin_addresses_and_balance_LATEST.tsv.gz
gunzip addresses.tsv.gz

# 3. Build
go build -o build/btc_recover ./cmd/btc_recover

# 4. Run recovery
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv \
    -checkpoint my_checkpoint.json \
    -c 5

# To stop: Press Ctrl+C
# To resume: Run the same command again!
```

## What You Get

### Features
- ✅ **Chits Mode**: Recover from scrambled word groups
- ✅ **Checkpoint/Resume**: Stop and continue anytime
- ✅ **Progress Reporting**: Live updates every N seconds
- ✅ **Multi-Address Types**: BIP44/49/84/86 (Legacy, SegWit, Taproot)
- ✅ **Fast Performance**: 50-100K mnemonics/sec
- ✅ **Match Logging**: Automatic save to `matches.log`

### CLI Flags Reference

| Flag | Example | Description |
|------|---------|-------------|
| `-chits` | (flag) | Enable chits mode |
| `-chits-file` | `my_chits.json` | Your scrambled words |
| `-addresses` | `addresses.tsv` | Known funded addresses |
| `-checkpoint` | `checkpoint.json` | Resume file (default: `chits_checkpoint.json`) |
| `-checkpoint-every` | `50000` | Checkpoint frequency (default: 100000) |
| `-i` | `20` | Address indexes to check (default: 20) |
| `-c` | `5` | Progress interval in seconds (default: 0=disabled) |
| `-v` | (flag) | Verbose output |

## Example Scenarios

### Scenario 1: 12-Word Mnemonic (3 Chits)

```json
{
  "chits": [
    ["word1", "word2", "word3", "word4"],
    ["word5", "word6", "word7", "word8"],
    ["word9", "word10", "word11", "word12"]
  ]
}
```

**Search Space**: 82,944 permutations  
**Expected Time**: 1-2 seconds  
**Success Rate**: High (if chits are correct)

### Scenario 2: 24-Word Mnemonic (6 Chits)

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

**Search Space**: 137 billion permutations  
**Expected Time**: ~1.5 days @ 100K/sec  
**Success Rate**: High (if chits are correct)  
**Tip**: Use checkpointing!

### Scenario 3: Uncertain About Few Words

If you're not 100% sure about some words, create chits with alternatives:

```json
{
  "chits": [
    ["word1", "word2"],
    ["word3_option1", "word3_option2"],
    ["word4", "word5"],
    ...
  ]
}
```

## Testing Your Setup

### 1. Run Quick Test
```bash
./test_quick.sh
```

Expected output:
```
✓ Checkpoint file created
✓ Resume test completed
All basic functionality working!
```

### 2. Run Benchmark
```bash
./benchmark_chits.sh
```

This will show you your system's throughput.

### 3. Verify Your Chits File

```bash
# Check if all words are valid BIP39 words
./build/btc_recover \
    -chits \
    -chits-file my_chits.json \
    -addresses addresses.tsv 2>&1 | head -20
```

Look for:
- ✅ "Chits Configuration" section showing your words
- ✅ "Total words: 12" or "Total words: 24"
- ❌ Any "invalid word" errors

## Common Issues

### "word X is not a valid BIP39 word"
**Solution**: Check spelling against [BIP39 wordlist](https://github.com/bitcoin/bips/blob/master/bip-0039/english.txt)

### "No matches found"
**Possible reasons**:
- Chits are incorrect
- Address not in TSV file
- Need to increase `-i` value
- Wrong address type (check derivation paths)

### Very slow performance
**Solutions**:
- Reduce `-i` if you know addresses are in early indexes
- Ensure TSV file is on SSD
- Check CPU usage with `top`

## Progress Monitoring

### Understanding Progress Output

```
Progress: 150000 total | 586 valid | 75000/sec
```

- **150000 total**: Mnemonics tested (includes invalid checksums)
- **586 valid**: Passed BIP39 checksum validation
- **75000/sec**: Testing throughput

### Checkpoint Output

```json
{
  "chit_perm_index": 2,
  "mnemonics_tested": 150000,
  "mnemonics_generated": 586,
  "addresses_checked": 46880,
  "timestamp": "2026-01-18T21:30:45Z"
}
```

This shows exactly where you can resume from.

## When You Find a Match

Output will show:
```
====================================================================
MATCH FOUND! Address: 1A... Type: p2pkh/0 Mnemonic: word1 word2...
====================================================================
```

Also saved to `matches.log`:
```
[2026-01-18T21:35:12Z] Address: 1A... | Type: p2pkh/0 | Mnemonic: ... | PrivKey: ...
```

**⚠️ IMPORTANT**: Immediately secure your mnemonic and delete all logs!

## Next Steps

1. **Read full documentation**: `CHITS_MODE.md`
2. **Performance tuning**: `BENCHMARK.md`
3. **Understand the code**: `INTEGRATION_SUMMARY.md`

## Support

Having issues? Check:
1. Logs in terminal output
2. Checkpoint file format (must be valid JSON)
3. Address file format (TSV with header)
4. System resources (memory for large address files)

## Example: Real Recovery Session

```bash
# My actual chits
cat > my_real_chits.json << EOF
{
  "chits": [
    ["abandon", "ability", "able", "about"],
    ["above", "absent", "absorb", "abstract"],
    ["absurd", "abuse", "access", "accident"]
  ]
}
EOF

# Start recovery
./build/btc_recover \
    -chits \
    -chits-file my_real_chits.json \
    -addresses addresses.tsv \
    -checkpoint recovery_checkpoint.json \
    -checkpoint-every 50000 \
    -c 5 \
    -v

# If interrupted, just run same command to resume!
```

---

**Ready to recover your wallet? Start with the Quick Test above! 🚀**
