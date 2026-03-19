package worker

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// ParseDerivationPaths parses comma-separated BIP purpose numbers
func ParseDerivationPaths(pathStr string) ([]uint32, error) {
	if pathStr == "" {
		return []uint32{44, 49, 84, 86}, nil // Default: all standard paths
	}

	parts := strings.Split(pathStr, ",")
	paths := make([]uint32, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		num, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid purpose number '%s': %w", part, err)
		}

		purpose := uint32(num)
		
		// Validate known purposes
		if purpose != 44 && purpose != 49 && purpose != 84 && purpose != 86 {
			log.Printf("Warning: BIP%d is not a standard Bitcoin derivation path", purpose)
		}

		paths = append(paths, purpose)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no valid derivation paths provided")
	}

	return paths, nil
}

// FormatDerivationPaths formats purposes for display
func FormatDerivationPaths(paths []uint32) string {
	names := make([]string, len(paths))
	for i, p := range paths {
		switch p {
		case 44:
			names[i] = fmt.Sprintf("BIP%d (Legacy P2PKH)", p)
		case 49:
			names[i] = fmt.Sprintf("BIP%d (P2SH-SegWit)", p)
		case 84:
			names[i] = fmt.Sprintf("BIP%d (Native SegWit)", p)
		case 86:
			names[i] = fmt.Sprintf("BIP%d (Taproot)", p)
		default:
			names[i] = fmt.Sprintf("BIP%d", p)
		}
	}
	return strings.Join(names, ", ")
}
