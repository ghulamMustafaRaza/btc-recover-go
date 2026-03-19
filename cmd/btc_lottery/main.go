package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"btc_recover/internal/lookup"
	"btc_recover/internal/worker"
)

var (
	// Data source
	addressFile = flag.String("addresses", "", "Path to TSV file with addresses (required)")

	// Mode selection
	chitsMode       = flag.Bool("chits", false, "Enable chits mode (partial mnemonic recovery)")
	chitsFile       = flag.String("chits-file", "", "Path to JSON file with chits (required for chits mode)")
	checkpointFile  = flag.String("checkpoint", "chits_checkpoint.json", "Path to checkpoint file for resume functionality")
	checkpointEvery = flag.Int64("checkpoint-every", 100000, "Save checkpoint every N tested mnemonics")
	matchesFile     = flag.String("matches-file", "matches.jsonl", "Path to JSON Lines file for storing matches")
	derivationPaths = flag.String("derivation-paths", "44,49,84,86", "Comma-separated BIP purpose numbers (e.g., '44,49,84' for Legacy,P2SH-SegWit,Native-SegWit)")

	// Worker configuration
	workers        = flag.Int("w", 32, "Number of CPU workers")
	addressIndexes = flag.Int("i", 20, "Number of address indexes to check per mnemonic (0-n)")
	entropyBits    = flag.Int("e", 128, "Entropy bits: 128 (12 words) or 256 (24 words)")

	// GPU configuration
	useGPU       = flag.Bool("gpu", false, "Enable GPU acceleration")
	gpuBatchSize = flag.Int("batch", 12500, "GPU batch size in mnemonics")
	ptxPath      = flag.String("ptx", "", "Path to btc_recover.ptx (auto-detect if not set)")
	gtableXPath  = flag.String("gtable-x", "", "Path to GTable X file (generate in memory if not set)")
	gtableYPath  = flag.String("gtable-y", "", "Path to GTable Y file (generate in memory if not set)")

	// Output configuration
	counterInterval = flag.Int("c", 0, "Interval for reporting address count (0 = disabled)")
	verbose         = flag.Bool("v", false, "Enable verbose output")

	// Notifications
	pushoverToken = flag.String("pt", "", "Pushover application token")
	pushoverUser  = flag.String("pu", "", "Pushover user key")

	// Mutex for file writes
	matchesFileMutex sync.Mutex
)

// workerConfig holds configuration for worker creation.
type workerConfig struct {
	numWorkers      int
	addressIndexes  int
	entropyBits     int
	gpuBatchSize    int
	useGPU          bool
	verbose         bool
	ptxPath         string
	gtableXPath     string
	gtableYPath     string
	chitsMode       bool
	chitsFile       string
	checkpointPath  string
	checkpointEvery int64
	derivationPaths string
}

func main() {
	flag.Parse()

	// Validate mode selection
	if *chitsMode {
		if *chitsFile == "" {
			log.Fatal("Chits mode requires -chits-file <path-to-json>")
		}
		log.Printf("BTC Recover v2 - Chits Recovery Mode")
	} else {
		// Validate entropy bits for random mode
		if *entropyBits != 128 && *entropyBits != 256 {
			log.Fatal("Entropy bits must be 128 (12 words) or 256 (24 words)")
		}
		mnemonicWords := *entropyBits / 32 * 3
		log.Printf("BTC Recover v2 - GPU Accelerated")
		log.Printf("Workers: %d, Address indexes: %d, Mnemonic: %d words", *workers, *addressIndexes, mnemonicWords)
	}

	if *addressFile == "" {
		log.Fatal("Must specify -addresses <path-to-tsv>")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load addresses into memory
	var hashSet *lookup.AddressHashSet
	var err error

	log.Printf("Loading addresses from %s...", *addressFile)
	hashSet, err = lookup.LoadFromTSV(lookup.LoadConfig{
		FilePath:         *addressFile,
		ProgressInterval: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to load addresses: %v", err)
	}

	log.Printf("Loaded %d addresses (%.1f MB memory)",
		hashSet.TotalAddresses(),
		float64(hashSet.MemoryUsage())/(1024*1024))

	// Display checkpoint info if it exists
	if *chitsMode && *checkpointFile != "" {
		if _, err := os.Stat(*checkpointFile); err == nil {
			log.Printf("Found existing checkpoint: %s", *checkpointFile)
			log.Printf("Recovery will resume from last saved position")
		}
	}

	// Worker configuration
	cfg := workerConfig{
		numWorkers:      *workers,
		addressIndexes:  *addressIndexes,
		entropyBits:     *entropyBits,
		gpuBatchSize:    *gpuBatchSize,
		useGPU:          *useGPU,
		verbose:         *verbose,
		ptxPath:         *ptxPath,
		gtableXPath:     *gtableXPath,
		gtableYPath:     *gtableYPath,
		chitsMode:       *chitsMode,
		chitsFile:       *chitsFile,
		checkpointPath:  *checkpointFile,
		checkpointEvery: *checkpointEvery,
		derivationPaths: *derivationPaths,
	}

	// Start workers
	var totalMatches int64
	matchChan, getStats, waitWorkers := runWorkers(ctx, hashSet, cfg)

	// Aggregate matches from all workers
	go func() {
		for match := range matchChan {
			atomic.AddInt64(&totalMatches, 1)
			logMatch(match)
		}
	}()

	// Progress reporter
	if *counterInterval > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(*counterInterval) * time.Second)
			defer ticker.Stop()

			lastCount := int64(0)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					current, mnemonics := getStats()
					rate := (current - lastCount) / int64(*counterInterval)
					lastCount = current

					var msg string
					if mnemonics > 0 {
						msg = fmt.Sprintf("Checked %d addresses (%d/sec), %d mnemonics", current, rate, mnemonics)
					} else {
						msg = fmt.Sprintf("Checked %d addresses (%d/sec)", current, rate)
					}
					log.Println(msg)

					if *pushoverToken != "" && *pushoverUser != "" {
						go sendPushoverNotification(*pushoverToken, *pushoverUser, "BTC Recover Progress", msg)
					}
				}
			}
		}()
	}

	// Wait for shutdown
	<-ctx.Done()
	log.Println("Shutdown signal received, waiting for workers to finish...")

	// Give workers time to finish
	done := make(chan struct{})
	go func() {
		waitWorkers()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers finished")
	case <-time.After(10 * time.Second):
		log.Println("Timeout waiting for workers")
	}

	final, _ := getStats()
	matches := atomic.LoadInt64(&totalMatches)
	log.Printf("Shutdown complete. Total addresses checked: %d, Matches found: %d", final, matches)

	if matches > 0 {
		log.Printf("Matches saved to: %s", *matchesFile)
		log.Printf("⚠️  SECURE YOUR RECOVERY PHRASE AND DELETE ALL OUTPUT FILES!")
	}
}

func logMatch(match worker.Match) {
	msg := fmt.Sprintf("MATCH FOUND! Address: %s Type: %s Mnemonic: %s",
		match.Address, match.AddrType, match.Mnemonic)

	// Print to console with emphasis
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(msg)
	fmt.Println(strings.Repeat("=", 60))

	// Save to JSON Lines file (mutex-protected)
	matchesFileMutex.Lock()
	saveMatchToJSON(match)
	matchesFileMutex.Unlock()

	// Also save to legacy text log for compatibility
	matchesFileMutex.Lock()
	file, err := os.OpenFile("matches.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		matchesFileMutex.Unlock()
		log.Printf("Error opening matches.log: %v", err)
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	logLine := fmt.Sprintf("[%s] Address: %s | Type: %s | Mnemonic: %s | PrivKey: %s\n",
		timestamp, match.Address, match.AddrType, match.Mnemonic, match.PrivateKey)
	if _, err := file.WriteString(logLine); err != nil {
		log.Printf("Error writing to matches.log: %v", err)
	}
	file.Close()
	matchesFileMutex.Unlock()

	// Send push notification
	if *pushoverToken != "" && *pushoverUser != "" {
		go sendPushoverNotification(*pushoverToken, *pushoverUser, "BTC RECOVER MATCH!", msg)
	}
}

// MatchRecord represents a match stored in JSON format
type MatchRecord struct {
	Timestamp      string `json:"timestamp"`
	Address        string `json:"address"`
	AddressType    string `json:"address_type"`
	DerivationPath string `json:"derivation_path"`
	Mnemonic       string `json:"mnemonic"`
	PrivateKey     string `json:"private_key"`
	PublicKey      string `json:"public_key"`
}

func saveMatchToJSON(match worker.Match) {
	file, err := os.OpenFile(*matchesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening %s: %v", *matchesFile, err)
		return
	}
	defer file.Close()

	// Parse address type to extract path info (e.g., "p2pkh/0" -> "m/44'/0'/0'/0/0")
	derivationPath := parseDerivationPath(match.AddrType)

	record := MatchRecord{
		Timestamp:      time.Now().Format(time.RFC3339),
		Address:        match.Address,
		AddressType:    match.AddrType,
		DerivationPath: derivationPath,
		Mnemonic:       match.Mnemonic,
		PrivateKey:     match.PrivateKey,
		PublicKey:      match.PublicKey,
	}

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(record); err != nil {
		log.Printf("Error encoding match to JSON: %v", err)
		return
	}

	log.Printf("Match saved to %s", *matchesFile)
}

func parseDerivationPath(addrType string) string {
	// addrType format: "p2pkh/0", "p2sh-p2wpkh/1", etc.
	parts := strings.Split(addrType, "/")
	if len(parts) != 2 {
		return "unknown"
	}

	typeStr := parts[0]
	index := parts[1]

	var purpose string
	switch typeStr {
	case "p2pkh":
		purpose = "44"
	case "p2sh-p2wpkh":
		purpose = "49"
	case "p2wpkh":
		purpose = "84"
	case "p2tr":
		purpose = "86"
	default:
		purpose = "?"
	}

	return fmt.Sprintf("m/%s'/0'/0'/0/%s", purpose, index)
}

func sendPushoverNotification(token, user, title, message string) error {
	form := url.Values{}
	form.Set("token", token)
	form.Set("user", user)
	form.Set("title", title)
	form.Set("message", message)

	req, err := http.NewRequest("POST", "https://api.pushover.net/1/messages.json", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response from Pushover: %s", resp.Status)
	}

	return nil
}
