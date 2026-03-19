package worker

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync/atomic"

	"btc_recover/internal/lookup"

	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/tyler-smith/go-bip39"
)

// ChitsConfig holds configuration for chits-based recovery.
type ChitsConfig struct {
	Chits           [][]string // Groups of words (e.g., 6 groups of 4 words for 24-word seed)
	AddressIndexes  int        // Number of address indexes to check per mnemonic
	DerivationPaths []uint32   // BIP purpose numbers (e.g., [44, 49, 84, 86])
	Verbose         bool
	CheckpointPath  string // Path to checkpoint file (empty to disable)
	CheckpointEvery int64  // Save checkpoint every N tested mnemonics
}

// ChitsWorker processes chits (partial word groups) to recover mnemonics.
// It generates permutations of chit orders and word orders within each chit.
type ChitsWorker struct {
	hashSet *lookup.AddressHashSet
	cfg     ChitsConfig

	addressesChecked   int64
	mnemonicsGenerated int64
	mnemonicsTested    int64
	matchesFound       int64

	// Checkpoint state
	currentChitPermIndex int
	lastCheckpointCount  int64
}

// NewChitsWorker creates a new chits-based worker.
func NewChitsWorker(hashSet *lookup.AddressHashSet, cfg ChitsConfig) *ChitsWorker {
	return &ChitsWorker{
		hashSet: hashSet,
		cfg:     cfg,
	}
}

// Run starts the worker loop, generating all permutations of chits.
func (w *ChitsWorker) Run(ctx context.Context) <-chan Match {
	matches := make(chan Match, 10)

	go func() {
		defer close(matches)

		// Try to load checkpoint
		var startChitPermIndex int
		if w.cfg.CheckpointPath != "" {
			if cp, err := LoadChitsCheckpoint(w.cfg.CheckpointPath); err == nil && cp != nil {
				startChitPermIndex = cp.ChitPermIndex
				w.mnemonicsGenerated = cp.MnemonicsGenerated
				w.mnemonicsTested = cp.MnemonicsTested
				w.addressesChecked = cp.AddressesChecked
				w.lastCheckpointCount = cp.MnemonicsTested
				log.Printf("Resuming from checkpoint: chit perm %d, %d mnemonics tested",
					startChitPermIndex, cp.MnemonicsTested)
			} else if err != nil && !os.IsNotExist(err) {
				log.Printf("Warning: Failed to load checkpoint: %v", err)
			}
		}

		// Generate all permutations of chits
		chitPerms := permutations(len(w.cfg.Chits))

		for i, chitOrder := range chitPerms {
			// Skip to checkpoint position
			if i < startChitPermIndex {
				continue
			}

			select {
			case <-ctx.Done():
				w.saveCheckpoint(i)
				return
			default:
			}

			w.currentChitPermIndex = i

			// For each chit order, generate all word permutations within each chit
			w.processChitOrder(ctx, chitOrder, matches)

			// Save checkpoint periodically
			if w.cfg.CheckpointPath != "" && w.cfg.CheckpointEvery > 0 {
				if w.mnemonicsTested-w.lastCheckpointCount >= w.cfg.CheckpointEvery {
					w.saveCheckpoint(i)
					w.lastCheckpointCount = w.mnemonicsTested
				}
			}
		}

		// Save final checkpoint
		if w.cfg.CheckpointPath != "" {
			w.saveCheckpoint(len(chitPerms))
		}

		if w.cfg.Verbose {
			log.Printf("ChitsWorker finished: %d mnemonics tested, %d valid, %d addresses checked",
				w.mnemonicsTested, w.mnemonicsGenerated, w.addressesChecked)
		}
	}()

	return matches
}

// saveCheckpoint saves the current progress.
func (w *ChitsWorker) saveCheckpoint(chitPermIndex int) {
	cp := ChitsCheckpoint{
		ChitPermIndex:      chitPermIndex,
		MnemonicsGenerated: atomic.LoadInt64(&w.mnemonicsGenerated),
		MnemonicsTested:    atomic.LoadInt64(&w.mnemonicsTested),
		AddressesChecked:   atomic.LoadInt64(&w.addressesChecked),
	}

	if err := SaveChitsCheckpoint(w.cfg.CheckpointPath, cp); err != nil {
		log.Printf("Warning: Failed to save checkpoint: %v", err)
	} else if w.cfg.Verbose {
		log.Printf("Checkpoint saved: chit perm %d, %d mnemonics tested",
			chitPermIndex, cp.MnemonicsTested)
	}
}

// processChitOrder processes a specific ordering of chits.
func (w *ChitsWorker) processChitOrder(ctx context.Context, chitOrder []int, matches chan<- Match) {
	// Get the number of words in each chit (they should all be the same)
	if len(w.cfg.Chits) == 0 || len(w.cfg.Chits[0]) == 0 {
		return
	}

	// Generate all word permutations for each chit in the order
	wordPermsByChit := make([][][]string, len(chitOrder))
	for i, chitIdx := range chitOrder {
		wordPermsByChit[i] = permutationsOfStrings(w.cfg.Chits[chitIdx])
	}

	// Generate cartesian product of all word permutations
	w.generateCombinations(ctx, wordPermsByChit, 0, []string{}, matches)
}

// generateCombinations recursively generates all combinations of word permutations.
func (w *ChitsWorker) generateCombinations(ctx context.Context, wordPermsByChit [][][]string, chitIdx int, current []string, matches chan<- Match) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if chitIdx == len(wordPermsByChit) {
		// We have a complete candidate mnemonic
		w.testMnemonic(ctx, current, matches)
		return
	}

	// Try each word permutation for the current chit
	for _, wordPerm := range wordPermsByChit[chitIdx] {
		newCurrent := make([]string, len(current)+len(wordPerm))
		copy(newCurrent, current)
		copy(newCurrent[len(current):], wordPerm)

		w.generateCombinations(ctx, wordPermsByChit, chitIdx+1, newCurrent, matches)
	}
}

// testMnemonic tests a candidate mnemonic for validity and address matches.
func (w *ChitsWorker) testMnemonic(ctx context.Context, words []string, matches chan<- Match) {
	atomic.AddInt64(&w.mnemonicsTested, 1)

	// Build mnemonic string
	mnemonicStr := ""
	for i, word := range words {
		if i > 0 {
			mnemonicStr += " "
		}
		mnemonicStr += word
	}

	// Validate BIP39 checksum
	mnemonic, err := bip39.NewMnemonic([]byte(mnemonicStr))
	if err != nil {
		// Alternative: try to parse as existing mnemonic
		entropy, err2 := bip39.EntropyFromMnemonic(mnemonicStr)
		if err2 != nil {
			return // Invalid mnemonic
		}
		mnemonic, err2 = bip39.NewMnemonic(entropy)
		if err2 != nil {
			return
		}
	}

	// It's a valid mnemonic!
	atomic.AddInt64(&w.mnemonicsGenerated, 1)

	// Generate seed and derive addresses
	seed := bip39.NewSeed(mnemonic, "")

	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		if w.cfg.Verbose {
			log.Printf("Error creating master key: %v", err)
		}
		return
	}

	// Pre-derive change keys for each address type
	changeKeys := make(map[uint32]*hdkeychain.ExtendedKey)
	for _, purpose := range w.cfg.DerivationPaths {
		changeKey, err := deriveChangeKeyHD(masterKey, purpose)
		if err != nil {
			if w.cfg.Verbose {
				log.Printf("Error deriving change key for purpose %d: %v", purpose, err)
			}
			return
		}
		changeKeys[purpose] = changeKey
	}

	// Generate all addresses
	addresses := make([]addressInfo, 0, len(w.cfg.DerivationPaths)*w.cfg.AddressIndexes)

	for idx := uint32(0); idx < uint32(w.cfg.AddressIndexes); idx++ {
		for _, purpose := range w.cfg.DerivationPaths {
			var addr addressInfo
			var err error

			switch purpose {
			case 44:
				// BIP44 - P2PKH (Legacy)
				addr, err = w.deriveP2PKHFromChangeHD(changeKeys[44], idx, mnemonicStr)
			case 49:
				// BIP49 - P2SH-P2WPKH (Wrapped SegWit)
				addr, err = w.deriveP2SHFromChangeHD(changeKeys[49], idx, mnemonicStr)
			case 84:
				// BIP84 - P2WPKH (Native SegWit)
				addr, err = w.deriveP2WPKHFromChangeHD(changeKeys[84], idx, mnemonicStr)
			case 86:
				// BIP86 - P2TR (Taproot)
				addr, err = w.deriveP2TRFromChangeHD(changeKeys[86], idx, mnemonicStr)
			default:
				if w.cfg.Verbose {
					log.Printf("Warning: Unsupported derivation path BIP%d", purpose)
				}
				continue
			}

			if err == nil {
				addresses = append(addresses, addr)
			} else if w.cfg.Verbose {
				log.Printf("Error deriving address for BIP%d index %d: %v", purpose, idx, err)
			}
		}
	}

	atomic.AddInt64(&w.addressesChecked, int64(len(addresses)))

	// Check all addresses against hash set
	addrStrings := make([]string, len(addresses))
	for i, a := range addresses {
		addrStrings[i] = a.address
	}

	found := w.hashSet.ContainsBatch(addrStrings)

	for _, addr := range addresses {
		if found[addr.address] {
			atomic.AddInt64(&w.matchesFound, 1)
			select {
			case matches <- Match{
				Address:    addr.address,
				PrivateKey: addr.privateKey,
				PublicKey:  addr.publicKey,
				Mnemonic:   addr.mnemonic,
				AddrType:   addr.addrType,
			}:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Stats returns current statistics.
func (w *ChitsWorker) Stats() Stats {
	return Stats{
		AddressesChecked:   atomic.LoadInt64(&w.addressesChecked),
		MnemonicsGenerated: atomic.LoadInt64(&w.mnemonicsGenerated),
		MatchesFound:       atomic.LoadInt64(&w.matchesFound),
	}
}

// Close releases resources.
func (w *ChitsWorker) Close() error {
	return nil
}

// Helper methods for address derivation (same as CPUWorker)

func (w *ChitsWorker) deriveP2PKHFromChangeHD(changeKey *hdkeychain.ExtendedKey, idx uint32, mnemonic string) (addressInfo, error) {
	childKey, err := changeKey.Derive(idx)
	if err != nil {
		return addressInfo{}, err
	}

	privKey, err := childKey.ECPrivKey()
	if err != nil {
		return addressInfo{}, err
	}

	wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		return addressInfo{}, err
	}

	pubKeyBytes := wif.SerializePubKey()
	pubKeyHash := btcutil.Hash160(pubKeyBytes)

	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
	if err != nil {
		return addressInfo{}, err
	}

	return addressInfo{
		address:    addr.EncodeAddress(),
		privateKey: wif.String(),
		publicKey:  hex.EncodeToString(pubKeyBytes),
		mnemonic:   mnemonic,
		addrType:   fmt.Sprintf("p2pkh/%d", idx),
	}, nil
}

func (w *ChitsWorker) deriveP2SHFromChangeHD(changeKey *hdkeychain.ExtendedKey, idx uint32, mnemonic string) (addressInfo, error) {
	childKey, err := changeKey.Derive(idx)
	if err != nil {
		return addressInfo{}, err
	}

	privKey, err := childKey.ECPrivKey()
	if err != nil {
		return addressInfo{}, err
	}

	wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		return addressInfo{}, err
	}

	pubKeyBytes := wif.SerializePubKey()
	pubKeyHash := btcutil.Hash160(pubKeyBytes)

	witnessProgram := append([]byte{0x00, 0x14}, pubKeyHash...)
	scriptHash := btcutil.Hash160(witnessProgram)

	addr, err := btcutil.NewAddressScriptHashFromHash(scriptHash, &chaincfg.MainNetParams)
	if err != nil {
		return addressInfo{}, err
	}

	return addressInfo{
		address:    addr.EncodeAddress(),
		privateKey: wif.String(),
		publicKey:  hex.EncodeToString(pubKeyBytes),
		mnemonic:   mnemonic,
		addrType:   fmt.Sprintf("p2sh-p2wpkh/%d", idx),
	}, nil
}

func (w *ChitsWorker) deriveP2WPKHFromChangeHD(changeKey *hdkeychain.ExtendedKey, idx uint32, mnemonic string) (addressInfo, error) {
	childKey, err := changeKey.Derive(idx)
	if err != nil {
		return addressInfo{}, err
	}

	privKey, err := childKey.ECPrivKey()
	if err != nil {
		return addressInfo{}, err
	}

	wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		return addressInfo{}, err
	}

	pubKeyBytes := wif.SerializePubKey()
	pubKeyHash := btcutil.Hash160(pubKeyBytes)

	addr, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
	if err != nil {
		return addressInfo{}, err
	}

	return addressInfo{
		address:    addr.EncodeAddress(),
		privateKey: wif.String(),
		publicKey:  hex.EncodeToString(pubKeyBytes),
		mnemonic:   mnemonic,
		addrType:   fmt.Sprintf("p2wpkh/%d", idx),
	}, nil
}

func (w *ChitsWorker) deriveP2TRFromChangeHD(changeKey *hdkeychain.ExtendedKey, idx uint32, mnemonic string) (addressInfo, error) {
	childKey, err := changeKey.Derive(idx)
	if err != nil {
		return addressInfo{}, err
	}

	privKey, err := childKey.ECPrivKey()
	if err != nil {
		return addressInfo{}, err
	}

	wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		return addressInfo{}, err
	}

	internalPubKey := privKey.PubKey()
	taprootKey := txscript.ComputeTaprootKeyNoScript(internalPubKey)

	addr, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(taprootKey), &chaincfg.MainNetParams)
	if err != nil {
		return addressInfo{}, err
	}

	return addressInfo{
		address:    addr.EncodeAddress(),
		privateKey: wif.String(),
		publicKey:  hex.EncodeToString(schnorr.SerializePubKey(internalPubKey)),
		mnemonic:   mnemonic,
		addrType:   fmt.Sprintf("p2tr/%d", idx),
	}, nil
}

// Utility functions for permutations

// permutations generates all permutations of indices [0..n-1].
func permutations(n int) [][]int {
	if n <= 0 {
		return [][]int{{}}
	}

	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}

	var result [][]int
	permute(indices, 0, &result)
	return result
}

// permute recursively generates permutations using Heap's algorithm.
func permute(arr []int, k int, result *[][]int) {
	if k == len(arr) {
		perm := make([]int, len(arr))
		copy(perm, arr)
		*result = append(*result, perm)
		return
	}

	for i := k; i < len(arr); i++ {
		arr[k], arr[i] = arr[i], arr[k]
		permute(arr, k+1, result)
		arr[k], arr[i] = arr[i], arr[k]
	}
}

// permutationsOfStrings generates all permutations of a string slice.
func permutationsOfStrings(strs []string) [][]string {
	if len(strs) == 0 {
		return [][]string{{}}
	}
	if len(strs) == 1 {
		return [][]string{strs}
	}

	var result [][]string
	permuteStrings(strs, 0, &result)
	return result
}

// permuteStrings recursively generates string permutations.
func permuteStrings(arr []string, k int, result *[][]string) {
	if k == len(arr) {
		perm := make([]string, len(arr))
		copy(perm, arr)
		*result = append(*result, perm)
		return
	}

	for i := k; i < len(arr); i++ {
		arr[k], arr[i] = arr[i], arr[k]
		permuteStrings(arr, k+1, result)
		arr[k], arr[i] = arr[i], arr[k]
	}
}
