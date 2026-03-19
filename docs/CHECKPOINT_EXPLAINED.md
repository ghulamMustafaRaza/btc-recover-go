# Checkpoint File Explained

## Understanding Your Checkpoint

When you see a checkpoint file like this:

```json
{
  "chit_perm_index": 0,
  "word_perm_indices": null,
  "mnemonics_generated": 234391,
  "mnemonics_tested": 60020945,
  "addresses_checked": 1875128,
  "timestamp": "2026-01-18T21:44:26.797678+05:00"
}
```

Here's what each field means and why it matters.

## Field-by-Field Breakdown

### 1. `chit_perm_index` - Which Chit Order Is Being Tested

**Value**: `0` in your case

**What it means**:
- Indicates which **ordering of chits** is currently being processed
- For 6 chits, there are **720 possible orderings** (6! = 6×5×4×3×2×1)
- `0` = first permutation (chits in the order you provided in the JSON file)
- `1` = second permutation (different order)
- ... up to `719` (last permutation)

**Your Progress**:
```
Chit Orderings:  [  0  ][ 1 ][ 2 ][ 3 ]...[ 719 ]
                  ^
                  You are here (still on first ordering)
```

**Why it matters for resume**:
- If you stop at `chit_perm_index: 42`, it will resume from permutation #43
- Prevents re-testing the same chit orders

**Example**:
If your chits are `[A, B, C, D, E, F]`:
- Index 0: `A, B, C, D, E, F` (original order)
- Index 1: `A, B, C, D, F, E` (last two swapped)
- Index 2: `A, B, C, E, D, F` (different arrangement)
- ... 717 more permutations ...

### 2. `word_perm_indices` - Word Order Within Chits

**Value**: `null` (not currently used)

**What it means**:
- Would track which word permutation within each chit is being tested
- Each chit of 4 words has 24 permutations (4!)
- Currently this is handled internally, not checkpointed

**Future enhancement**:
Could be used for more granular resume:
```json
{
  "word_perm_indices": [12, 5, 18, 0, 9, 3]
}
```
Meaning:
- Chit 1: on word permutation 12/24
- Chit 2: on word permutation 5/24
- etc.

This would allow resuming within a chit ordering, but adds complexity.

### 3. `mnemonics_generated` - Valid BIP39 Seeds Found

**Value**: `234,391` in your case

**What it means**:
- Number of **valid BIP39 mnemonics** created (passed checksum)
- From all the word combinations tested, only these had valid checksums
- This is the actual number of mnemonics that had addresses derived

**Why it's much smaller than mnemonics_tested**:
- BIP39 uses the last word (or part of it) as a checksum
- For 24-word mnemonics: only **1 in 256** combinations is valid
- For 12-word mnemonics: only **1 in 16** combinations is valid

**Your numbers**:
```
mnemonics_tested:    60,020,945
mnemonics_generated:    234,391
Ratio: 234,391 / 60,020,945 = 0.0039 = 1/256 ✓ (correct!)
```

**What happens to the valid ones**:
1. Seed is generated via PBKDF2
2. Addresses are derived (BIP32/BIP44/49/84/86)
3. Each address is checked against your address list
4. If match → saved to `matches.jsonl`

### 4. `mnemonics_tested` - Total Permutations Tried

**Value**: `60,020,945` in your case

**What it means**:
- **Total word combinations** attempted (valid + invalid)
- This is your main progress indicator
- Includes all permutations that failed BIP39 checksum

**How it's calculated**:
For each chit ordering:
- 4 words per chit = 24 permutations (4!)
- 6 chits = 24^6 = 191,102,976 total word permutations
- Your progress: 60M / 191M = **31.4% of first chit ordering**

**Full search space**:
```
Total for 6 chits of 4 words:
= 720 chit orderings × 191,102,976 word perms
= 137,594,142,720 total permutations
= ~137.6 billion

Your progress: 60M / 137.6B = 0.044%
```

**Speed estimate**:
At 100,000 mnemonics/sec:
- Time to complete one chit ordering: 191M / 100K = 1,911 seconds ≈ **32 minutes**
- Time for all 720 orderings: 720 × 32 min = **16 days**

### 5. `addresses_checked` - Addresses Derived and Searched

**Value**: `1,875,128` in your case

**What it means**:
- Total Bitcoin addresses generated and checked against your address list
- Each valid mnemonic generates multiple addresses

**How it's calculated**:
```
addresses_checked = mnemonics_generated × addresses_per_mnemonic
```

**Addresses per mnemonic** depends on your settings:
- With `-i 20` (default): 20 indexes × 4 types = **80 addresses**
- With `-i 10`: 10 indexes × 4 types = **40 addresses**
- With `-i 2`: 2 indexes × 4 types = **8 addresses**

**Your calculation**:
```
1,875,128 / 234,391 = 8 addresses per mnemonic
```
This means you used `-i 2` (checking only first 2 indexes)

**The 4 address types checked**:
1. **BIP44** (m/44'/0'/0'/0/i) - Legacy P2PKH (starts with "1...")
2. **BIP49** (m/49'/0'/0'/0/i) - P2SH-SegWit (starts with "3...")
3. **BIP84** (m/84'/0'/0'/0/i) - Native SegWit (starts with "bc1q...")
4. **BIP86** (m/86'/0'/0'/0/i) - Taproot (starts with "bc1p...")

### 6. `timestamp` - When Checkpoint Was Saved

**Value**: `"2026-01-18T21:44:26.797678+05:00"`

**What it means**:
- ISO 8601 timestamp of when checkpoint was last saved
- Includes timezone offset (+05:00)

**Uses**:
- Calculate how long recovery has been running
- Track checkpoint freshness
- Audit trail

## Summary of Your Progress

Based on your checkpoint file:

```
╔═══════════════════════════════════════════════════════════╗
║                   Recovery Progress                       ║
╚═══════════════════════════════════════════════════════════╝

Current Chit Ordering:     0 / 720  (0.14%)
Word Combinations Tested:  60,020,945 / 191,102,976  (31.4% of current ordering)
Valid BIP39 Found:         234,391  (expected ~234k, ✓ correct)
Addresses Checked:         1,875,128
Address Indexes Per Path:  2  (checking 0-1 for each address type)

Overall Progress:          60M / 137.6B  (0.044%)
Estimated Remaining:       ~16 days at 100K mnemonics/sec
```

## Why No Matches Yet?

If you haven't found matches (no `matches.jsonl` file), it could be:

1. **Wrong Chit Order**
   - You're still on chit ordering #0
   - The correct order might be #42 or #500 or any of the other 719 permutations
   
2. **Not Far Enough**
   - Only 31% through the first ordering
   - Need to complete all word permutations for this chit order

3. **Wrong Address List**
   - The addresses you're searching for aren't in your TSV file
   - Or they're at higher indexes (> 1)

4. **Wrong Chits**
   - The words in your chits file don't form the correct mnemonic

## When Will Matches Be Found?

**The `matches.jsonl` file is created ONLY when a match is found.**

If the correct mnemonic is:
- In the **current chit order** (index 0) → will find in ~32 min total
- In **later chit order** (e.g., index 500) → need to wait longer

To increase chances:
1. **Increase address indexes**: Use `-i 20` instead of `-i 2`
2. **Verify chits are correct**: All words must be from the actual mnemonic
3. **Check address file**: Ensure target addresses are in the TSV
4. **Be patient**: 137 billion permutations take time!

## Monitoring Progress

### View Current State
```bash
cat chits_checkpoint.json | jq '.'
```

### Calculate Progress Percentage
```bash
TESTED=$(cat chits_checkpoint.json | jq '.mnemonics_tested')
TOTAL=137594142720  # For 24-word with 6 chits
PERCENT=$(echo "scale=4; $TESTED * 100 / $TOTAL" | bc)
echo "Progress: ${PERCENT}%"
```

### Estimate Time Remaining
```bash
# Get rate from progress output (e.g., 100000/sec)
RATE=100000
TESTED=$(cat chits_checkpoint.json | jq '.mnemonics_tested')
TOTAL=137594142720
REMAINING=$((TOTAL - TESTED))
SECONDS=$((REMAINING / RATE))
HOURS=$(echo "scale=1; $SECONDS / 3600" | bc)
echo "Estimated remaining: ${HOURS} hours"
```

## Understanding Match Files

When a match IS found, you'll see TWO files:

### 1. `matches.jsonl` (JSON Lines)
```json
{
  "timestamp": "2026-01-18T21:47:22+05:00",
  "address": "31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE",
  "address_type": "p2sh-p2wpkh/0",
  "derivation_path": "m/49'/0'/0'/0/0",
  "mnemonic": "word1 word2 word3...",
  "private_key": "L5eC...",
  "public_key": "03b1..."
}
```

### 2. `matches.log` (Text)
```
[2026-01-18T21:47:22+05:00] Address: 31qT... | Type: p2sh-p2wpkh/0 | Mnemonic: word1... | PrivKey: L5eC...
```

**If neither file exists**: No matches have been found yet.

## Troubleshooting

### Problem: Progress seems stuck

**Check**:
```bash
# Watch for changes
watch -n 5 'cat chits_checkpoint.json | jq ".mnemonics_tested"'
```

If number increases → working!  
If stuck → process may have stopped

### Problem: Very slow progress

**Solutions**:
1. Check CPU usage: `top -p $(pgrep btc_recover)`
2. Reduce address indexes: `-i 5` instead of `-i 20`
3. Verify not I/O bound: Move address file to SSD

### Problem: Want to skip to different chit ordering

**Manual checkpoint edit** (advanced):
```bash
# Backup first!
cp chits_checkpoint.json chits_checkpoint_backup.json

# Jump to ordering #100
cat chits_checkpoint.json | jq '.chit_perm_index = 100' > chits_checkpoint_new.json
mv chits_checkpoint_new.json chits_checkpoint.json

# Resume
./build/btc_recover -chits -chits-file ... -checkpoint chits_checkpoint.json ...
```

## See Also

- [CHECKPOINT_RESUME.md](CHECKPOINT_RESUME.md) - Full checkpoint guide
- [CHITS_MODE.md](CHITS_MODE.md) - Chits recovery documentation
- [QUICK_START.md](QUICK_START.md) - Getting started
