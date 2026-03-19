# BTC Recover

Attempt to recover the seed phrase for specific cryptocurrency accounts after the stored written backup of the seed phrase proved incomplete/defective and a successful recovery has not yet been possible.

## Performance

| Metric | Value |
|--------|-------|
| Throughput | **~160,000 addresses/sec** (32 workers) |
| Memory usage | ~2.5 GB (for 50M addresses) |
| Time per mnemonic | ~1.6ms (80 addresses) |

Tested on AMD Ryzen 9 9300X3D with 64GB RAM.

## What It Does

1. Loads ~50 million known funded Bitcoin addresses into memory as a sorted hash set
2. Generates random BIP39 mnemonics (12 or 24 words)
3. Derives addresses using standard BIP derivation paths
4. Checks generated addresses against the in-memory hash set (O(log n) binary search)
5. Logs any matches (there won't be any)

## Address Types Generated

For each mnemonic, the program derives multiple address types across multiple indexes:

| Type | BIP | Path | Format | Example Prefix |
|------|-----|------|--------|----------------|
| P2PKH | BIP44 | `m/44'/0'/0'/0/i` | Legacy | `1...` |
| P2SH-P2WPKH | BIP49 | `m/49'/0'/0'/0/i` | Wrapped SegWit | `3...` |
| P2WPKH | BIP84 | `m/84'/0'/0'/0/i` | Native SegWit | `bc1q...` |
| P2TR | BIP86 | `m/86'/0'/0'/0/i` | Taproot | `bc1p...` |

With default settings (20 indexes), each mnemonic produces **80 addresses** (4 types × 20 indexes).

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Main Process                            │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              In-Memory Address Hash Set                  │   │
│  │  • 50M addresses as sorted 8-byte hash prefixes (~400MB)│   │
│  │  • O(log n) binary search (~26 comparisons max)         │   │
│  │  • Full address strings for match verification (~1.7GB) │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              ↑                                  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐       ┌──────────┐    │
│  │ Worker 1 │ │ Worker 2 │ │ Worker 3 │  ...  │ Worker N │    │
│  │          │ │          │ │          │       │          │    │
│  │ Generate │ │ Generate │ │ Generate │       │ Generate │    │
│  │ Mnemonic │ │ Mnemonic │ │ Mnemonic │       │ Mnemonic │    │
│  │    ↓     │ │    ↓     │ │    ↓     │       │    ↓     │    │
│  │ PBKDF2   │ │ PBKDF2   │ │ PBKDF2   │       │ PBKDF2   │    │
│  │    ↓     │ │    ↓     │ │    ↓     │       │    ↓     │    │
│  │ Derive   │ │ Derive   │ │ Derive   │       │ Derive   │    │
│  │ 80 addrs │ │ 80 addrs │ │ 80 addrs │       │ 80 addrs │    │
│  │    ↓     │ │    ↓     │ │    ↓     │       │    ↓     │    │
│  │ Check    │ │ Check    │ │ Check    │       │ Check    │    │
│  └──────────┘ └──────────┘ └──────────┘       └──────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### Key Optimizations

1. **In-Memory Hash Set**: Replaces PostgreSQL + Bloom filter with sorted hash array for O(log n) lookups
2. **hdkeychain Library**: Uses btcd's optimized BIP32 implementation (48x faster than go-bip32)
3. **Cached Derivation Paths**: Pre-computes hardened derivation paths (m/purpose'/0'/0'/0) once per mnemonic
4. **Concurrent Workers**: Independent goroutines with no lock contention

## Modes of Operation

### 1. Random Mode (Default)
Generates random mnemonics to demonstrate the impossibility of guessing Bitcoin keys.

### 2. Chits Mode (Purpose built) 🎯
Recovers wallets from partial mnemonic information. If you remember your seed phrase in groups but not the order, this mode can help!

**Key Features:**
- ✅ Checkpoint/Resume: Stop and continue anytime
- ✅ JSON Configuration: No code changes needed
- ✅ Fast: 50-100K mnemonics/sec
- ✅ Multi-address types: BIP44/49/84/86

**Quick Links:**
- 📖 [QUICK_START.md](QUICK_START.md) - Get started in 5 minutes
- 📚 [CHITS_MODE.md](CHITS_MODE.md) - Complete documentation
- 💾 [CHECKPOINT_RESUME.md](CHECKPOINT_RESUME.md) - Checkpoint & Resume guide
- 🎯 [BENCHMARK.md](BENCHMARK.md) - Performance guide
- 🔧 [INTEGRATION_SUMMARY.md](INTEGRATION_SUMMARY.md) - Technical details
- ✅ [TEST_VALIDATION.md](TEST_VALIDATION.md) - Test cases

## Quick Start

### Random Mode
```bash
# 1. Download address data (~2GB compressed)
curl -L -o addresses.tsv.gz \
  http://addresses.loyce.club/blockchair_bitcoin_addresses_and_balance_LATEST.tsv.gz
gunzip addresses.tsv.gz

# 2. Build
go build -o build/btc_recover ./cmd/btc_recover

# 3. Run
./build/btc_recover -addresses addresses.tsv -c 5
```

### Chits Mode
```bash
# 1. Create a chits file (see chits_example.json)
# 2. Run with chits mode
./build/btc_recover -chits -chits-file my_chits.json -addresses addresses.tsv -c 5
```

See [CHITS_MODE.md](CHITS_MODE.md) for detailed documentation.

## Benchmarking

To test performance:

```bash
./benchmark_chits.sh
```

This will test the chits recovery mode with a small 12-word mnemonic and measure throughput. See [BENCHMARK.md](BENCHMARK.md) for detailed benchmarking guide.

## Building

```bash
# Standard build
go build -o build/btc_recover ./cmd/btc_recover

# Optimized build
go build -ldflags="-s -w" -o build/btc_recover ./cmd/btc_recover

# With GPU support (requires CUDA toolkit)
go build -tags cuda -o build/btc_recover_gpu ./cmd/btc_recover
```

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-addresses` | | Path to TSV file with addresses (required) |
| `-chits` | false | Enable chits mode (partial mnemonic recovery) |
| `-chits-file` | | Path to JSON file with chits (required for chits mode) |
| `-checkpoint` | chits_checkpoint.json | Path to checkpoint file for resume functionality |
| `-checkpoint-every` | 100000 | Save checkpoint every N tested mnemonics |
| `-matches-file` | matches.jsonl | Path to JSON Lines file for storing matches |
| `-derivation-paths` | 44,49,84,86 | Comma-separated BIP purpose numbers (e.g., '44,49,84') |
| `-w` | 32 | Number of concurrent workers |
| `-i` | 20 | Address indexes per mnemonic (0 to n-1) |
| `-e` | 128 | Entropy bits: 128 (12 words) or 256 (24 words) |
| `-c` | 0 | Progress report interval in seconds (0 = disabled) |
| `-v` | false | Verbose output |
| `-gpu` | false | Enable GPU acceleration |
| `-batch` | 12500 | GPU batch size in mnemonics |
| `-pt` | | Pushover application token |
| `-pu` | | Pushover user key |

## Example Output

```
2026/01/07 15:34:03 BTC Recover v2 - GPU Accelerated
2026/01/07 15:34:03 Workers: 32, Address indexes: 20, Mnemonic: 12 words
2026/01/07 15:34:03 Loading addresses from addresses.tsv...
2026/01/07 15:34:45 Loaded 50000000 addresses (2.1 GB memory)
2026/01/07 15:34:50 Starting 32 CPU workers...
2026/01/07 15:34:55 Checked 1732560 addresses (346512/sec), 21684 mnemonics
2026/01/07 15:35:00 Checked 3498560 addresses (353200/sec), 43759 mnemonics
```

## Project Structure

```
btc_recover/
├── cmd/btc_recover/       # Main application entry point
│   ├── main.go            # CLI and orchestration
│   ├── run_cpu.go         # CPU worker management
│   └── run_gpu.go         # GPU worker management (build tag: cuda)
├── internal/
│   ├── lookup/            # In-memory address hash set
│   │   ├── hashset.go     # Sorted hash array with binary search
│   │   └── loader.go      # TSV file loader
│   └── worker/            # Worker implementations
│       ├── interface.go   # Common types
│       ├── cpu_worker.go  # CPU-based address generation
│       └── gpu_worker.go  # GPU-accelerated (CUDA)
├── gpu/                   # GPU support (optional)
│   ├── cuda/              # CUDA kernels
│   ├── gtable/            # Precomputed EC points generator
│   └── wrapper/           # Go-CUDA interface
└── testdata/              # Sample data for testing
```

## GPU Support

The codebase includes experimental GPU acceleration using CUDA. With the hdkeychain optimization, CPU performance is now excellent (~160k addresses/sec), so GPU is optional.

To build with GPU support:

```bash
# Compile CUDA kernels (requires nvcc)
cd gpu/cuda && make

# Generate GTable (precomputed EC points)
go run ./cmd/gengtable -o gpu/cuda/

# Build with cuda tag
go build -tags cuda -o build/btc_recover_gpu ./cmd/btc_recover
```

## Data Source

Address data sourced from [Blockchair.com dumps](https://blockchair.com/dumps) via [addresses.loyce.club](http://addresses.loyce.club/).

The TSV file should have two columns: `address` and `balance` (tab-separated with header).

## Technical Details

### Why hdkeychain?

The original go-bip32 library computed EC public keys for fingerprints on every child derivation (~1ms each). With 80 addresses per mnemonic, this added 80+ unnecessary EC operations.

btcd's hdkeychain uses lazy public key computation, reducing per-mnemonic time from 112ms to 2.3ms (48x improvement).

### Memory Layout

| Component | Memory |
|-----------|--------|
| Address hash prefixes (50M × 8 bytes) | ~400 MB |
| Full address strings (50M × ~34 bytes) | ~1.7 GB |
| Working buffers | ~400 MB |
| **Total** | ~2.5 GB |