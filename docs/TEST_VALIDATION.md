# Test Validation

## Verified Working Test Case

The example chits file contains a **real, working test case** that can be used to verify the implementation.

### Test Mnemonic

When the chits are in the correct order (as provided in `chits_example.json`), they form this valid 24-word BIP39 mnemonic:

```
argue payment reveal filter dice please sponsor clip choose choice melody shallow identify palace bone common jar nest cool lunar neck coffee differ plate
```

### Derived Addresses

This mnemonic derives to the following addresses (at index 0):

| Path | Type | Address |
|------|------|---------|
| m/44'/0'/0'/0/0 | Legacy (P2PKH) | `1PbJ41eNd1dqVeYLaFgxDUC7BgWk3RQZ6d` |
| m/49'/0'/0'/0/0 | P2SH-SegWit | `31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE` |
| m/84'/0'/0'/0/0 | Native SegWit | `bc1qcm7kznj4rlcs73u59r44vn7v5r0z98vvyslsgc` |

### Quick Verification Test

```bash
# This should find matches IMMEDIATELY (within 1 second)
./build/btc_recover \
    -chits \
    -chits-file chits_example.json \
    -addresses test_addresses.tsv \
    -i 2 \
    -v
```

**Expected Result:**
- ✅ Match found at **first permutation** (chits already in correct order)
- ✅ Finds 3 addresses at index 0 and 1
- ✅ Total time: < 1 second

### Full Benchmark

```bash
./benchmark_chits.sh
```

This runs the full test including:
- Building from source
- Testing with correct order (finds immediately)
- Testing checkpoint/resume
- Performance analysis

### Verification Script

To manually verify the mnemonic:

```bash
./verify_test_case.sh
```

This will:
1. Check BIP39 validity
2. Derive addresses for multiple paths
3. Show which addresses match the expected test addresses

## Test Results

### Expected Output

```
Starting chits recovery worker...
============================================================
MATCH FOUND! Address: 1PbJ41eNd1dqVeYLaFgxDUC7BgWk3RQZ6d Type: p2pkh/0
============================================================
============================================================
MATCH FOUND! Address: 31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE Type: p2sh-p2wpkh/0
============================================================
============================================================
MATCH FOUND! Address: 36Z9MvdvpndEJmzZMKH6xTMVTnU7od6TpK Type: p2sh-p2wpkh/1
============================================================
```

### Performance Metrics

On modern hardware, when chits are in correct order:
- **Time to first match**: < 100ms
- **Total permutations tested**: 1 (correct order)
- **Addresses checked**: 80 (20 indexes × 4 address types)
- **Result**: SUCCESS ✅

### Testing Scrambled Chits

To test the full permutation engine, scramble the chits order in the JSON file:

```json
{
  "chits": [
    ["jar", "nest", "cool", "lunar"],           // Was 5th
    ["argue", "payment", "reveal", "filter"],   // Was 1st
    ["neck", "coffee", "differ", "plate"],      // Was 6th
    ["choose", "choice", "melody", "shallow"],  // Was 3rd
    ["identify", "palace", "bone", "common"],   // Was 4th
    ["dice", "please", "sponsor", "clip"]       // Was 2nd
  ]
}
```

The recovery should still find the correct mnemonic, but will take longer as it tries different permutations.

## Troubleshooting

### No matches found

If the test doesn't find matches, check:

1. **Chits file format**: Must be valid JSON
2. **Address file format**: TSV with header `address\tbalance`
3. **Address indexes**: Use `-i 2` to check first 2 indexes
4. **Paths**: Program checks BIP44/49/84/86 by default

### Matches found but different addresses

If you modify the chits and get different results:
- This is expected! Different word order = different mnemonic = different addresses
- Only the exact order shown forms the test mnemonic

### Performance issues

If the test is slow:
- Ensure you're using the compiled binary (`./build/btc_recover`)
- Check CPU usage
- Verify the chits are in correct order (should find immediately)

## Integration with Rust Version

The test case matches the Rust implementation:

**Rust (`src/chits.rs`):**
```rust
pub fn get_chits() -> Vec<Vec<String>> {
    vec![
        vec!["argue".to_string(), "payment".to_string(), "reveal".to_string(), "filter".to_string()],
        vec!["dice".to_string(), "please".to_string(), "sponsor".to_string(), "clip".to_string()],
        // ... (same words)
    ]
}
```

**Go (`chits_example.json`):**
```json
{
  "chits": [
    ["argue", "payment", "reveal", "filter"],
    ["dice", "please", "sponsor", "clip"],
    // ... (same words)
  ]
}
```

Both implementations use the same test mnemonic and should find the same addresses.

## Security Warning

⚠️ **DO NOT USE THIS MNEMONIC FOR REAL FUNDS!**

This is a publicly known test mnemonic used for validation purposes only. Any funds sent to these addresses can be stolen by anyone with access to this code.

## Automated Testing

The test suite automatically validates:
- ✅ BIP39 validity
- ✅ Checksum verification  
- ✅ Address derivation
- ✅ Lookup correctness
- ✅ Match detection
- ✅ Checkpoint/resume

Run all tests:
```bash
./test_quick.sh          # Quick validation
./benchmark_chits.sh     # Full benchmark
./verify_test_case.sh    # Manual verification
```
