# Why No `matches.jsonl` File?

## Quick Answer

The `matches.jsonl` file is **only created when matches are found**. 

Based on your situation, you have `matches.log` (text format) but no `matches.jsonl` because:

1. The matches in `matches.log` were from **benchmark tests** 
2. **JSON Lines support was added later** in the implementation
3. Your **actual recovery run** (with 60M mnemonics tested) hasn't found matches yet

## Your Current Situation

### What You Have

```bash
$ ls -lh *.log *.json
chits_checkpoint.json   # Your progress: 60M mnemonics tested
matches.log             # 18 matches from benchmark tests
```

### What the Checkpoint Shows

```json
{
  "chit_perm_index": 0,          // Still on first chit ordering
  "mnemonics_tested": 60020945,  // 60M tested (31% of first ordering)
  "mnemonics_generated": 234391, // Valid BIP39 found
  "addresses_checked": 1875128   // Addresses checked
}
```

**Translation**: You're 31% through the **first chit ordering** out of 720 possible orderings.

## Two Types of Files

### 1. Benchmark Test Matches (Old)

Your `matches.log` contains **test matches** from running the benchmark:

```
[2026-01-18T21:33:36] Address: 1PbJ41... | Mnemonic: argue payment reveal...
[2026-01-18T21:33:36] Address: 31qT6X... | Mnemonic: argue payment reveal...
```

These are from the **example test case** (`chits_example.json`) that has the correct order.

### 2. Your Real Recovery (In Progress)

Your actual recovery with different chits:
- **Started**: Unknown (checkpoint timestamp: 2026-01-18T21:44:26)
- **Progress**: 60M / 137.6B permutations (0.044%)
- **Matches found**: **NONE YET** (that's why no `matches.jsonl`)

## When Will `matches.jsonl` Be Created?

The file will be created **immediately** when the first match is found:

```bash
# Run recovery
./build/btc_recover -chits -chits-file my_chits.json ...

# When match found:
============================================================
MATCH FOUND! Address: 31qT... Type: p2sh-p2wpkh/0
============================================================
2026-01-18 22:00:00 Match saved to matches.jsonl  # <-- File created here
```

## Understanding Your Progress

### Visual Representation

```
Total Search Space (6 chits, 24 words):
┌─────────────────────────────────────────────────────────┐
│ 720 Chit Orderings                                      │
├─────────────────────────────────────────────────────────┤
│ [0]────────────────────→ 191M word permutations         │
│  │                                                       │
│  └─ 60M tested (31% complete) ← YOU ARE HERE            │
│     └─ 234k valid BIP39                                 │
│        └─ 1.8M addresses checked                        │
│           └─ 0 matches found (yet)                      │
│                                                          │
│ [1] ......... (not started)                             │
│ [2] ......... (not started)                             │
│ ...                                                      │
│ [719] ....... (not started)                             │
└─────────────────────────────────────────────────────────┘
```

### Why No Matches Yet?

**Possible reasons**:

1. **Wrong chit order** 
   - Correct order might be permutation #234 or #567
   - You're still on #0

2. **Need to finish current ordering**
   - Only 31% through first permutation
   - Correct words might be in remaining 69%

3. **Wrong address indexes**
   - Using `-i 2` (only checking indexes 0-1)
   - Address might be at index 5 or 15

4. **Different chits than test**
   - Test matches are from `chits_example.json`
   - Your actual recovery uses different words

## What's in matches.log?

These are **test matches** from the working example:

```bash
$ cat matches.log
# 18 matches total - all from the same test mnemonic:
# "argue payment reveal filter dice please sponsor clip..."

# At different addresses and derivation paths:
- 1PbJ... (p2pkh/0)     # m/44'/0'/0'/0/0
- 31qT... (p2sh-p2wpkh/0) # m/49'/0'/0'/0/0
- 36Z9... (p2sh-p2wpkh/1) # m/49'/0'/0'/0/1
... (more addresses)
```

These demonstrate the system works correctly!

## Next Steps

### Option 1: Continue Current Recovery

Your recovery is making progress. To monitor:

```bash
# Check progress every 5 seconds
watch -n 5 'cat chits_checkpoint.json | jq "{tested: .mnemonics_tested, valid: .mnemonics_generated, ordering: .chit_perm_index}"'
```

**Expected timeline** (at 100K mnemonics/sec):
- Complete first ordering: ~32 minutes total
- Test all 720 orderings: ~16 days

### Option 2: Test with Known Working Example

To see JSON Lines in action:

```bash
# Use the working test case
./build/btc_recover \
    -chits \
    -chits-file chits_example.json \
    -addresses test_addresses.tsv \
    -matches-file NEW_matches.jsonl \
    -checkpoint NEW_checkpoint.json \
    -i 2

# Should find matches immediately and create:
# - NEW_matches.jsonl (JSON format) ✓
# - NEW_checkpoint.json (progress) ✓
```

### Option 3: Verify Your Setup

Run the validation:

```bash
./test_checkpoint_resume.sh
```

This will:
- Create matches in JSONL format
- Test checkpoint/resume
- Verify everything works

## How to Check for Matches

### During Recovery

Watch the console output:

```bash
# You'll see this when found:
============================================================
MATCH FOUND! Address: 31qT... Type: p2sh-p2wpkh/0
============================================================
Match saved to matches.jsonl
```

### After Recovery

```bash
# Check if file exists
ls -lh matches.jsonl

# Count matches
wc -l matches.jsonl

# View matches
cat matches.jsonl | jq '.'

# Extract mnemonic
cat matches.jsonl | jq -r '.mnemonic' | head -1
```

## Comparing the Two Formats

### Old (Text) - matches.log
```
[2026-01-18T21:33:36+05:00] Address: 31qT... | Type: p2sh-p2wpkh/0 | Mnemonic: argue... | PrivKey: L3kz...
```

**Pros**: Human-readable  
**Cons**: Hard to parse programmatically

### New (JSON) - matches.jsonl
```json
{
  "timestamp": "2026-01-18T21:47:22+05:00",
  "address": "31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE",
  "address_type": "p2sh-p2wpkh/0",
  "derivation_path": "m/49'/0'/0'/0/0",
  "mnemonic": "argue payment reveal filter...",
  "private_key": "L3kzbNydMWiKsUTwKVZGRFW86bZPvGGPtbSTwfdRVSfAi4V5ZHSd",
  "public_key": "031240259f8dab8adbc96cb24a47258900cb417876b99dec48b72420c54ca5def3"
}
```

**Pros**: 
- Easy to parse with `jq`, Python, etc.
- Includes derivation path
- One complete object per line
- Machine-readable

**Cons**: More verbose

**Both files are created** for every match (backward compatibility).

## Summary

| Question | Answer |
|----------|--------|
| **Why no matches.jsonl?** | No matches found in your actual recovery yet |
| **What's in matches.log?** | 18 test matches from benchmark runs |
| **Is recovery working?** | Yes! 60M mnemonics tested, checkpointing working |
| **When will matches.jsonl exist?** | When first match is found |
| **How much progress?** | 0.044% of total search space |
| **Should I be worried?** | No! This is normal for large search spaces |

## Bottom Line

Your recovery is **working correctly**:
- ✅ Checkpoint saving
- ✅ Progress tracking
- ✅ Address checking

You just haven't found the right word combination yet. With 137 billion permutations, finding the correct one can take time!

**The `matches.jsonl` file will be created automatically when you find a match.** 🎯
