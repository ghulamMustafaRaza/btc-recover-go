#!/bin/bash

# Verify if the example chits form a valid mnemonic and derive the expected address

set -e

cd "$(dirname "$0")"
cd ..  # Go to project root

echo "Verifying Test Case"
echo "==================="
echo ""

# The test address from Rust code
TEST_ADDRESS="31qT6Xr7778kMHKBRQKAvjarsdnG9XaQXE"

# Concatenate chits in order to form the mnemonic
MNEMONIC="argue payment reveal filter dice please sponsor clip choose choice melody shallow identify palace bone common jar nest cool lunar neck coffee differ plate"

echo "Test mnemonic (24 words):"
echo "$MNEMONIC"
echo ""
echo "Expected address: $TEST_ADDRESS"
echo ""

# Create a test Go program to verify
cat > /tmp/verify_mnemonic.go << 'GOEOF'
package main

import (
	"fmt"
	"os"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/tyler-smith/go-bip39"
)

func main() {
	mnemonic := os.Args[1]
	
	// Check if valid BIP39
	_, err := bip39.EntropyFromMnemonic(mnemonic)
	if err != nil {
		fmt.Printf("❌ Invalid BIP39 mnemonic: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("✓ Valid BIP39 mnemonic")
	
	// Generate seed
	seed := bip39.NewSeed(mnemonic, "")
	
	// Derive master key
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		fmt.Printf("❌ Error creating master key: %v\n", err)
		os.Exit(1)
	}
	
	// Test BIP49 (P2SH-SegWit) path m/49'/0'/0'/0/0
	// This derives to addresses starting with "3..."
	purposeKey, _ := masterKey.Derive(hdkeychain.HardenedKeyStart + 49)
	coinKey, _ := purposeKey.Derive(hdkeychain.HardenedKeyStart + 0)
	accountKey, _ := coinKey.Derive(hdkeychain.HardenedKeyStart + 0)
	changeKey, _ := accountKey.Derive(0)
	
	fmt.Println("\nDerived addresses for first 5 indexes:")
	fmt.Println("Path: m/49'/0'/0'/0/i (P2SH-SegWit, starts with '3...')")
	fmt.Println("")
	
	for i := uint32(0); i < 5; i++ {
		childKey, _ := changeKey.Derive(i)
		privKey, _ := childKey.ECPrivKey()
		wif, _ := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
		
		pubKeyBytes := wif.SerializePubKey()
		pubKeyHash := btcutil.Hash160(pubKeyBytes)
		witnessProgram := append([]byte{0x00, 0x14}, pubKeyHash...)
		scriptHash := btcutil.Hash160(witnessProgram)
		
		addr, _ := btcutil.NewAddressScriptHashFromHash(scriptHash, &chaincfg.MainNetParams)
		fmt.Printf("  [%d] %s\n", i, addr.EncodeAddress())
	}
	
	// Also test other paths
	fmt.Println("\nPath: m/44'/0'/0'/0/i (Legacy, starts with '1...')")
	purposeKey, _ = masterKey.Derive(hdkeychain.HardenedKeyStart + 44)
	coinKey, _ = purposeKey.Derive(hdkeychain.HardenedKeyStart + 0)
	accountKey, _ = coinKey.Derive(hdkeychain.HardenedKeyStart + 0)
	changeKey, _ = accountKey.Derive(0)
	
	for i := uint32(0); i < 5; i++ {
		childKey, _ := changeKey.Derive(i)
		privKey, _ := childKey.ECPrivKey()
		wif, _ := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
		
		pubKeyBytes := wif.SerializePubKey()
		pubKeyHash := btcutil.Hash160(pubKeyBytes)
		addr, _ := btcutil.NewAddressPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
		fmt.Printf("  [%d] %s\n", i, addr.EncodeAddress())
	}
}
GOEOF

# Compile and run
cd /tmp
export GO111MODULE=on
go mod init verify_mnemonic 2>/dev/null || true
go get github.com/btcsuite/btcd/btcutil@latest 2>/dev/null || true
go get github.com/btcsuite/btcd/btcutil/hdkeychain@latest 2>/dev/null || true
go get github.com/tyler-smith/go-bip39@latest 2>/dev/null || true

go build -o verify_mnemonic verify_mnemonic.go

echo "Checking mnemonic validity and deriving addresses..."
echo ""

./verify_mnemonic "$MNEMONIC"

echo ""
echo "Expected test address: $TEST_ADDRESS"
echo ""
echo "❓ Does any of the above addresses match? If not, the test case needs a real mnemonic."
