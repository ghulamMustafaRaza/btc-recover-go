# Chits Mode - Partial Mnemonic Recovery

## Overview

Chits Mode is a powerful feature that allows you to recover a Bitcoin wallet when you have **partial information** about your mnemonic phrase. Instead of randomly generating mnemonics, it systematically tests all permutations of known word groups ("chits") to find the correct mnemonic.

## What Are Chits?

**Chits** are groups of words from your mnemonic phrase that you remember, but you don't know the correct order. For example, if you have a 24-word mnemonic but only remember it in groups of 4 words each:

- Chit 1: `["argue", "payment", "reveal", "filter"]`
- Chit 2: `["dice", "please", "sponsor", "clip"]`
- Chit 3: `["choose", "choice", "melody", "shallow"]`
- Chit 4: `["identify", "palace", "bone", "common"]`
- Chit 5: `["jar", "nest", "cool", "lunar"]`
- Chit 6: `["neck", "coffee", "differ", "plate"]`

The program will:
1. Try all possible **orderings of chits** (e.g., Chit 2 → Chit 5 → Chit 1 → ...)
2. For each ordering, try all **permutations of words within each chit**
3. Validate the BIP39 checksum
4. Derive addresses from valid mnemonics
5. Check if any generated address matches your known addresses

## Use Cases

### Scenario 1: Scrambled Word Groups
You wrote down your 24-word mnemonic in groups of 4, but the papers got shuffled and you don't know which group comes first.

### Scenario 2: Partial Memory
You remember most words but not their exact positions. You can group words that you think are close together.

### Scenario 3: Multiple Possibilities
You have several candidate words for certain positions and want to test all combinations.

## Quick Start

### 1. Create a Chits File

Create a JSON file (e.g., `my_chits.json`) with your word groups:

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

**Important:** 
- All words must be valid BIP39 words (from the English wordlist)
- For a 12-word mnemonic, use 3 chits of 4 words each
- For a 24-word mnemonic, use 6 chits of 4 words each (or adjust as needed)

### 2. Prepare Address Data

Download a list of funded Bitcoin addresses (same as regular mode):

```bash
curl -L -o addresses.tsv.gz \
  http://addresses.loyce.club/blockchair_bitcoin_addresses_and_balance_LATEST.tsv.gz
gunzip addresses.tsv.gz
```

### 3. Run Chits Mode

```bash
./build/btc_recover -chits -chits-file my_chits.json -addresses addresses.tsv -i 20 -c 5
```

### CLI Flags for Chits Mode

| Flag | Required | Description |
|------|----------|-------------|
| `-chits` | Yes | Enable chits mode |
| `-chits-file` | Yes | Path to JSON file with chits |
| `-addresses` | Yes | Path to TSV file with known addresses |
| `-i` | No | Address indexes to check per mnemonic (default: 20) |
| `-c` | No | Progress report interval in seconds (default: 0) |
| `-v` | No | Verbose output (default: false) |

## Search Space Analysis

The program automatically calculates and displays the search space before starting:

### Example: 6 Chits of 4 Words Each (24-word mnemonic)

- **Chit orderings:** 6! = 720
- **Word permutations per chit:** 4! = 24
- **Total word permutations:** 24^6 = 191,102,976
- **Total permutations:** 720 × 191,102,976 = **137,594,142,720**
- **Expected valid mnemonics** (after BIP39 checksum): ~537,477,901

At 100,000 mnemonics/sec, this would take approximately **1.5 days**.

### Example: 3 Chits of 4 Words Each (12-word mnemonic)

- **Chit orderings:** 3! = 6
- **Word permutations per chit:** 4! = 24
- **Total word permutations:** 24^3 = 13,824
- **Total permutations:** 6 × 13,824 = **82,944**
- **Expected valid mnemonics** (after BIP39 checksum): ~5,184

At 100,000 mnemonics/sec, this would take approximately **less than 1 second**.

## Example Output

```
BTC Recover v2 - Chits Recovery Mode
Loading addresses from addresses.tsv...
Loaded 50000000 addresses (2.1 GB memory)

╔════════════════════════════════════════════════════╗
║              Chits Configuration                   ║
╚════════════════════════════════════════════════════╝

  Chit 1: [argue payment reveal filter]
  Chit 2: [dice please sponsor clip]
  Chit 3: [choose choice melody shallow]
  Chit 4: [identify palace bone common]
  Chit 5: [jar nest cool lunar]
  Chit 6: [neck coffee differ plate]

Total words: 24

Search Space Analysis:
  • Total permutations: 137594142720
  • Expected valid mnemonics: ~537477901 (after BIP39 checksum)
  • Estimated time (100k/sec): 1.5 days

Starting chits recovery worker...
  Progress: 1000000 total | 3906 valid | 200000/sec
  Progress: 2000000 total | 7812 valid | 200000/sec
  ...
```

## Performance Tips

1. **Reduce Address Indexes**: If you know your addresses are in the first few indexes, use `-i 5` instead of `-i 20`
2. **Use Smaller Chits**: If you're certain about some word positions, create chits with just 2-3 words
3. **Test Incrementally**: Start with a small subset of addresses to verify your chits are correct
4. **Monitor Progress**: Use `-c 5` to see progress updates every 5 seconds

## Important Notes

### Checksum Validation
The BIP39 checksum significantly reduces the search space. Only ~1/256 of all permutations will pass validation for 24-word mnemonics (or ~1/16 for 12-word mnemonics).

### Word Order Within Chits
If you're confident about the order of words **within** each chit, you can modify the code to skip word permutations and only test chit orderings. This would reduce the search space from 137 billion to just 720 permutations for the example above!

### Match Logging
When a match is found, it's automatically logged to `matches.log` with:
- Timestamp
- Full mnemonic phrase
- Matched address
- Address type (p2pkh, p2wpkh, etc.)
- Private key (WIF format)

**⚠️ SECURITY WARNING:** Delete `matches.log` and any output files after recovery!

## Comparison with Random Mode

| Feature | Random Mode | Chits Mode |
|---------|------------|------------|
| **Use Case** | Testing impossibility of guessing | Recovering known partial mnemonic |
| **Success Probability** | ~0% (2^128 keyspace) | High (if chits are correct) |
| **Speed** | 160k addresses/sec | 100-200k mnemonics/sec |
| **Search Space** | Infinite | Finite and calculable |
| **Time to Success** | Never (statistically) | Minutes to days (depending on chits) |

## Troubleshooting

### "Invalid chits: word X is not a valid BIP39 word"
Check the [BIP39 wordlist](https://github.com/bitcoin/bips/blob/master/bip-0039/english.txt) and verify your words are spelled correctly.

### "No matches found"
Possible reasons:
- Chits are incorrect or incomplete
- Address not in the loaded TSV file
- Not checking enough address indexes (try increasing `-i`)
- Wrong derivation path (program checks BIP44/49/84/86)

### Very Long Estimated Time
Consider:
- Are all chits necessary? Can you be certain about some positions?
- Can you reduce the number of words per chit?
- Are you using the correct word groupings?

## Technical Implementation

The chits worker:
1. Generates all permutations of chit orderings using Heap's algorithm
2. For each ordering, generates all word permutations within each chit
3. Tests each candidate mnemonic for BIP39 validity
4. Derives addresses using `hdkeychain` (fast BIP32 implementation)
5. Checks addresses against the in-memory hash set (O(log n) binary search)

## Example Files

See `chits_example.json` in the repository for a complete example.

## Contributing

If you successfully recover a wallet using this tool (even a test wallet), please consider sharing your success story (anonymously) to help others!