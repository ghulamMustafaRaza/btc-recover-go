package chits

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tyler-smith/go-bip39"
)

// ChitsData represents the structure of chits loaded from a file.
type ChitsData struct {
	Chits [][]string `json:"chits"`
}

// LoadFromFile loads chits from a JSON file.
// Expected format:
//
//	{
//	  "chits": [
//	    ["argue", "payment", "reveal", "filter"],
//	    ["dice", "please", "sponsor", "clip"],
//	    ...
//	  ]
//	}
func LoadFromFile(path string) ([][]string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var data ChitsData
	if err := json.Unmarshal(file, &data); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	if len(data.Chits) == 0 {
		return nil, fmt.Errorf("no chits found in file")
	}

	return data.Chits, nil
}

// ValidateChits validates that all words in chits are valid BIP39 words.
func ValidateChits(chits [][]string) error {
	wordlist := bip39.GetWordList()
	wordSet := make(map[string]bool)
	for _, word := range wordlist {
		wordSet[word] = true
	}

	for i, group := range chits {
		for j, word := range group {
			// Normalize to lowercase
			normalized := strings.ToLower(strings.TrimSpace(word))
			if !wordSet[normalized] {
				return fmt.Errorf("chit[%d][%d]: '%s' is not a valid BIP39 word", i, j, word)
			}
			// Update with normalized word
			chits[i][j] = normalized
		}
	}

	return nil
}

// CalculateSearchSpace calculates the total number of permutations to test.
func CalculateSearchSpace(chits [][]string) (totalPerms, expectedValid int64) {
	if len(chits) == 0 {
		return 0, 0
	}

	// Calculate factorial
	factorial := func(n int) int64 {
		if n <= 1 {
			return 1
		}
		result := int64(1)
		for i := 2; i <= n; i++ {
			result *= int64(i)
		}
		return result
	}

	// Chit permutations (e.g., 6! for 6 chits)
	chitPerms := factorial(len(chits))

	// Word permutations within each chit
	wordPermsPerChit := int64(1)
	for _, group := range chits {
		wordPermsPerChit *= factorial(len(group))
	}

	totalPerms = chitPerms * wordPermsPerChit

	// BIP39 checksum is last word's last 4/8 bits depending on mnemonic length
	// For 12 words: 128 bits entropy + 4 bits checksum (1/16 valid)
	// For 24 words: 256 bits entropy + 8 bits checksum (1/256 valid)
	totalWords := 0
	for _, group := range chits {
		totalWords += len(group)
	}

	if totalWords == 12 {
		expectedValid = totalPerms / 16
	} else if totalWords == 24 {
		expectedValid = totalPerms / 256
	} else {
		// Unknown mnemonic length, estimate conservatively
		expectedValid = totalPerms / 256
	}

	return totalPerms, expectedValid
}

// DisplayInfo prints information about the loaded chits.
func DisplayInfo(chits [][]string) {
	fmt.Println("\n╔════════════════════════════════════════════════════╗")
	fmt.Println("║              Chits Configuration                   ║")
	fmt.Println("╚════════════════════════════════════════════════════╝")
	fmt.Println()

	totalWords := 0
	for i, group := range chits {
		totalWords += len(group)
		fmt.Printf("  Chit %d: %v\n", i+1, group)
	}

	fmt.Printf("\nTotal words: %d\n", totalWords)

	totalPerms, expectedValid := CalculateSearchSpace(chits)

	fmt.Println("\nSearch Space Analysis:")
	fmt.Printf("  • Total permutations: %d\n", totalPerms)
	fmt.Printf("  • Expected valid mnemonics: ~%d (after BIP39 checksum)\n", expectedValid)

	// Estimate time at 100k mnemonics/sec
	estimatedSeconds := float64(totalPerms) / 100000.0
	estimatedMinutes := estimatedSeconds / 60.0
	estimatedHours := estimatedMinutes / 60.0

	if estimatedHours < 1 {
		fmt.Printf("  • Estimated time (100k/sec): %.1f minutes\n", estimatedMinutes)
	} else if estimatedHours < 24 {
		fmt.Printf("  • Estimated time (100k/sec): %.1f hours\n", estimatedHours)
	} else {
		fmt.Printf("  • Estimated time (100k/sec): %.1f days\n", estimatedHours/24.0)
	}

	fmt.Println()
}
