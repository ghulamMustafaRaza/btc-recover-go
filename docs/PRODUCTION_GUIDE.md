# Production Recovery Guide

This guide explains how to use the production recovery script for real-world seed phrase recovery.

## Quick Start

```bash
./execution/recover.sh
```

That's it! The script will guide you through the entire process.

## What the Script Does

### Step 1: Setup
- Creates `execution/prod/` directory if needed
- Generates a sample `chits.json` file

### Step 2: Configure Chits
- Pauses and asks you to edit `execution/prod/chits.json`
- Replace the sample words with your actual word groups
- Press ENTER when done

### Step 3: Provide Addresses
- Waits for you to create `execution/prod/addresses.tsv`
- Continuously checks until the file appears
- Validates the file has content

### Step 4: Auto-Configuration
- Builds the latest binary
- Detects GPU hardware (NVIDIA)
- Checks for CUDA toolkit
- Enables GPU acceleration automatically if available

### Step 5: Recovery
- Shows configuration summary
- If checkpoint exists, offers to resume
- Starts the recovery process
- Saves progress automatically

## File Structure

All production files are stored in `execution/prod/`:

```
execution/prod/
├── chits.json              # Your word groups (input)
├── addresses.tsv           # Target addresses (input)
├── checkpoint.json         # Resume checkpoint (auto-saved)
├── matches.jsonl          # Found matches (output)
├── matches.log            # Found matches text log (output)
└── recovery_output.log    # Full recovery log (output)
```

## Preparing Your Chits File

Edit `execution/prod/chits.json`:

### For 24-word mnemonic (6 chits):
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

### For 12-word mnemonic (3 chits):
```json
{
  "chits": [
    ["word1", "word2", "word3", "word4"],
    ["word5", "word6", "word7", "word8"],
    ["word9", "word10", "word11", "word12"]
  ]
}
```

**Important**: Each chit must have exactly 4 words!

## Preparing Your Addresses File

Create `execution/prod/addresses.tsv` (tab-separated):

```
address	balance
1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa	50
1BvBMSEYstWetqTFn5Au4m4GFg7xJaNVN2	100
bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh	25
```

**Format**:
- Line 1: `address[TAB]balance` (header)
- Line 2+: `<bitcoin_address>[TAB]<amount>`
- Use actual TAB character (not spaces)

**Tips**:
- The balance column is optional (can be 0)
- More addresses = lower chance of false matches
- Addresses can be any format (Legacy, SegWit, Taproot)

## Recovery Process

### Starting Recovery
```bash
./execution/recover.sh
```

### Stopping Recovery
Press `Ctrl+C` at any time. Progress is saved automatically.

### Resuming Recovery
Just run the script again:
```bash
./execution/recover.sh
```

It will automatically detect the checkpoint and resume from where it left off.

### Monitoring Progress
The script shows real-time progress:
- Mnemonics tested
- Valid mnemonics found
- Addresses checked
- Current rate (mnemonics/sec)

### If a Match is Found
The script will:
1. Display the match immediately
2. Save to `execution/prod/matches.jsonl` (JSON format)
3. Save to `execution/prod/matches.log` (text format)
4. Continue searching for more matches

## GPU Acceleration

### Requirements
- NVIDIA GPU (GTX/RTX series)
- CUDA Toolkit installed

### How It Works
The script automatically:
1. Detects NVIDIA GPU using `nvidia-smi`
2. Checks for CUDA toolkit (`nvcc`)
3. Enables GPU mode if both are present
4. Falls back to CPU if not available

### Installing CUDA (if needed)
- Download from: https://developer.nvidia.com/cuda-downloads
- Choose your OS and follow instructions
- Restart terminal after installation

### Checking GPU Status
```bash
nvidia-smi  # Show GPU info
nvcc --version  # Show CUDA version
```

## Configuration

The script uses these optimized settings:

| Setting | Value | Description |
|---------|-------|-------------|
| Derivation Paths | 44,49,84,86 | All standard Bitcoin paths |
| Address Indexes | 20 | Check first 20 addresses per path |
| Checkpoint Frequency | 100,000 | Save every 100k mnemonics |
| Workers | 10 | Concurrent processing threads |
| GPU Batch Size | 1024 | GPU batch size (if GPU available) |

## Search Space

### 24-word mnemonic with 6 chits:
- Chit permutations: **720** (6!)
- Word permutations per chit: **24** (4!)
- Total word permutations: **191,102,976** (24^6)
- **Total combinations**: **~137 billion**
- **Valid mnemonics** (after checksum): **~537 million**

### 12-word mnemonic with 3 chits:
- Chit permutations: **6** (3!)
- Word permutations per chit: **24** (4!)
- Total word permutations: **13,824** (24^3)
- **Total combinations**: **~82,944**
- **Valid mnemonics** (after checksum): **~5,184**

### Time Estimates

**CPU Mode** (~1,000 mnemonics/sec):
- 12-word: ~5 seconds
- 24-word: ~149 hours (~6 days)

**GPU Mode** (~100,000 mnemonics/sec):
- 12-word: <1 second
- 24-word: ~1.5 hours

*Note: Times vary based on hardware and chit complexity*

## Troubleshooting

### "addresses.tsv not found"
- Script will wait until you create the file
- Place it in `execution/prod/addresses.tsv`
- Make sure it's tab-separated

### "No GPU detected"
- Script will use CPU mode automatically
- Check `nvidia-smi` to verify GPU
- Install CUDA toolkit for GPU support

### "Build failed"
- Ensure Go is installed: `go version`
- Check for compilation errors in output
- Required: Go 1.22 or later

### "No matches found"
- Check that addresses are correct
- Verify chits contain correct words
- Try increasing address indexes: edit script line with `-i 20` to `-i 50`
- Ensure chit order might be wrong (script tries all permutations)

### Recovery is slow
- Enable GPU acceleration (see GPU section)
- Reduce address indexes if checking too many
- Reduce derivation paths if you know which type you used

## Security Notes

1. **Private Keys**: Found matches contain private keys - keep them secure!
2. **Matches File**: `matches.jsonl` contains sensitive data - protect it
3. **Log Files**: Output logs may contain partial information - handle carefully
4. **Cleanup**: After recovery, consider deleting checkpoint and logs:
   ```bash
   rm -rf execution/prod/
   ```

## Advanced Usage

### Custom Derivation Paths
Edit the script line:
```bash
CMD="$CMD -derivation-paths 44,49,84,86"
```

Change to only check specific paths:
- `44` = Legacy (P2PKH) - addresses starting with `1`
- `49` = SegWit (P2SH) - addresses starting with `3`
- `84` = Native SegWit (Bech32) - addresses starting with `bc1q`
- `86` = Taproot (P2TR) - addresses starting with `bc1p`

### Custom Address Indexes
Edit the script line:
```bash
CMD="$CMD -i 20"
```

Change `20` to:
- `10` = Faster, might miss addresses
- `50` = Slower, more thorough
- `100` = Very slow, extremely thorough

### Checkpoint Frequency
Edit the script line:
```bash
CMD="$CMD -checkpoint-every 100000"
```

- Lower value = More frequent saves, slightly slower
- Higher value = Less frequent saves, slightly faster

## Getting Help

- Check output logs: `execution/prod/recovery_output.log`
- Review checkpoint: `execution/prod/checkpoint.json`
- Test with benchmark first: `./execution/benchmark_chits.sh`

## Example Session

```bash
$ ./execution/recover.sh

╔════════════════════════════════════════════════════╗
║     BTC Recover - Production Recovery Tool        ║
╚════════════════════════════════════════════════════╝

1. Setting up production environment...
✓ Production directory exists: execution/prod

2. Chits file already exists: execution/prod/chits.json

3. Addresses file already exists: execution/prod/addresses.tsv
  Found 4 lines (including header)

4. Building btc_recover...
✓ Build successful: build/btc_recover

5. Detecting GPU support...
✓ NVIDIA GPU detected!
NVIDIA GeForce RTX 3080, 10240 MiB
✓ CUDA detected: version 12.1
╔════════════════════════════════════════════════════╗
║           Recovery Configuration                   ║
╚════════════════════════════════════════════════════╝

Chits file:        execution/prod/chits.json
Addresses file:    execution/prod/addresses.tsv
Checkpoint file:   execution/prod/checkpoint.json
Matches file:      execution/prod/matches.jsonl
Output log:        execution/prod/recovery_output.log
GPU acceleration:  ENABLED

...recovery continues...
```

## Success!

When a match is found, you'll see:

```
╔════════════════════════════════════════════════════╗
║              MATCHES FOUND!!!                      ║
╚════════════════════════════════════════════════════╝

Total matches: 1
Matches file:  execution/prod/matches.jsonl

{
  "address": "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
  "private_key": "L3kg...",
  "public_key": "04...",
  "mnemonic": "word1 word2 word3 ...",
  "address_type": "P2PKH",
  "derivation_path": "m/44'/0'/0'/0/0"
}
```

**Your Bitcoin is recovered!** 🎉

Use the mnemonic or private key to access your funds in any Bitcoin wallet.
