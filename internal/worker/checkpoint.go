package worker

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ChitsCheckpoint represents the state of chits recovery for resuming.
type ChitsCheckpoint struct {
	ChitPermIndex       int       `json:"chit_perm_index"`       // Index of current chit permutation
	WordPermIndices     []int     `json:"word_perm_indices"`     // Indices for each chit's word permutation
	MnemonicsGenerated  int64     `json:"mnemonics_generated"`
	MnemonicsTested     int64     `json:"mnemonics_tested"`
	AddressesChecked    int64     `json:"addresses_checked"`
	Timestamp           time.Time `json:"timestamp"`
}

// SaveChitsCheckpoint saves the current state to a file.
func SaveChitsCheckpoint(path string, cp ChitsCheckpoint) error {
	cp.Timestamp = time.Now()
	
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling checkpoint: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing checkpoint file: %w", err)
	}

	return nil
}

// LoadChitsCheckpoint loads checkpoint from a file.
func LoadChitsCheckpoint(path string) (*ChitsCheckpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No checkpoint exists
		}
		return nil, fmt.Errorf("reading checkpoint file: %w", err)
	}

	var cp ChitsCheckpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("parsing checkpoint: %w", err)
	}

	return &cp, nil
}
