//go:build cuda

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"btc_recover/internal/chits"
	"btc_recover/internal/lookup"
	"btc_recover/internal/worker"
)

// runWorkers starts workers (GPU-enabled build).
func runWorkers(ctx context.Context, hashSet *lookup.AddressHashSet, cfg workerConfig) (matchChan chan worker.Match, statsFn func() (int64, int64), waitFn func()) {
	matchChan = make(chan worker.Match, 100)
	var totalAddresses int64
	var totalMnemonics int64
	var wg sync.WaitGroup

	// Check if chits mode is enabled
	if cfg.chitsMode {
		return runChitsWorkers(ctx, hashSet, cfg)
	}

	if cfg.useGPU {
		// Find PTX path
		ptxPath := cfg.ptxPath
		if ptxPath == "" {
			// Try common locations
			candidates := []string{
				"gpu/cuda/btc_recover.ptx",
				"/home/mo/dev/src/btc_recover/gpu/cuda/btc_recover.ptx",
				filepath.Join(filepath.Dir(os.Args[0]), "btc_recover.ptx"),
			}
			for _, p := range candidates {
				if _, err := os.Stat(p); err == nil {
					ptxPath = p
					break
				}
			}
			if ptxPath == "" {
				log.Fatal("Cannot find btc_recover.ptx. Use -ptx flag to specify path.")
			}
		}

		gpuCfg := worker.GPUWorkerConfig{
			Config: worker.Config{
				AddressIndexes: cfg.addressIndexes,
				EntropyBits:    cfg.entropyBits,
				GPUBatchSize:   cfg.gpuBatchSize,
				UseGPU:         true,
				Verbose:        cfg.verbose,
			},
			PTXPath:     ptxPath,
			GTableXPath: cfg.gtableXPath,
			GTableYPath: cfg.gtableYPath,
		}

		log.Printf("Creating GPU worker...")
		gpuWorker, err := worker.NewGPUWorker(hashSet, gpuCfg)
		if err != nil {
			log.Printf("Failed to create GPU worker: %v", err)
			log.Printf("Falling back to CPU workers")
			return runCPUWorkers(ctx, hashSet, cfg, matchChan, &totalAddresses, &wg)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer gpuWorker.Close()

			workerMatches := gpuWorker.Run(ctx)
			for match := range workerMatches {
				matchChan <- match
			}

			stats := gpuWorker.Stats()
			atomic.AddInt64(&totalAddresses, stats.AddressesChecked)
			atomic.AddInt64(&totalMnemonics, stats.MnemonicsGenerated)
		}()

		// Start CPU workers in parallel
		// GPU batches keys while CPU workers also check independently
		cpuWorkers := cfg.numWorkers - 1
		if cpuWorkers > 0 {
			log.Printf("Starting %d additional CPU workers...", cpuWorkers)
			for i := 0; i < cpuWorkers; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					workerCfg := worker.Config{
						AddressIndexes: cfg.addressIndexes,
						EntropyBits:    cfg.entropyBits,
						GPUBatchSize:   cfg.gpuBatchSize,
						UseGPU:         false,
						Verbose:        cfg.verbose,
					}

					w := worker.NewCPUWorker(hashSet, workerCfg)
					defer w.Close()

					workerMatches := w.Run(ctx)
					for match := range workerMatches {
						matchChan <- match
					}

					stats := w.Stats()
					atomic.AddInt64(&totalAddresses, stats.AddressesChecked)
				}(i)
			}
		}

		// Create a stats function that also checks GPU worker
		statsFn = func() (int64, int64) {
			gpuStats := gpuWorker.Stats()
			cpuAddr := atomic.LoadInt64(&totalAddresses)
			return cpuAddr + gpuStats.AddressesChecked, gpuStats.MnemonicsGenerated
		}

		waitFn = func() {
			wg.Wait()
			close(matchChan)
		}

		return
	}

	// Non-GPU mode
	return runCPUWorkers(ctx, hashSet, cfg, matchChan, &totalAddresses, &wg)
}

func runCPUWorkers(ctx context.Context, hashSet *lookup.AddressHashSet, cfg workerConfig, matchChan chan worker.Match, totalAddresses *int64, wg *sync.WaitGroup) (chan worker.Match, func() (int64, int64), func()) {
	workerCfg := worker.Config{
		AddressIndexes: cfg.addressIndexes,
		EntropyBits:    cfg.entropyBits,
		GPUBatchSize:   cfg.gpuBatchSize,
		UseGPU:         false,
		Verbose:        cfg.verbose,
	}

	// Keep references to workers for real-time stats
	workers := make([]*worker.CPUWorker, cfg.numWorkers)

	log.Printf("Starting %d CPU workers...", cfg.numWorkers)
	for i := 0; i < cfg.numWorkers; i++ {
		w := worker.NewCPUWorker(hashSet, workerCfg)
		workers[i] = w

		wg.Add(1)
		go func(w *worker.CPUWorker) {
			defer wg.Done()
			defer w.Close()

			workerMatches := w.Run(ctx)
			for match := range workerMatches {
				matchChan <- match
			}
		}(w)
	}

	// Real-time stats from all workers
	statsFn := func() (int64, int64) {
		var totalAddr, totalMnem int64
		for _, w := range workers {
			stats := w.Stats()
			totalAddr += stats.AddressesChecked
			totalMnem += stats.MnemonicsGenerated
		}
		return totalAddr, totalMnem
	}

	waitFn := func() {
		wg.Wait()
		close(matchChan)
	}

	return matchChan, statsFn, waitFn
}

// runChitsWorkers starts chits recovery workers.
func runChitsWorkers(ctx context.Context, hashSet *lookup.AddressHashSet, cfg workerConfig) (matchChan chan worker.Match, statsFn func() (int64, int64), waitFn func()) {
	matchChan = make(chan worker.Match, 100)
	var wg sync.WaitGroup

	// Load chits from file
	chitsData, err := chits.LoadFromFile(cfg.chitsFile)
	if err != nil {
		log.Fatalf("Failed to load chits: %v", err)
	}

	// Validate chits
	if err := chits.ValidateChits(chitsData); err != nil {
		log.Fatalf("Invalid chits: %v", err)
	}

	// Display chits information
	chits.DisplayInfo(chitsData)

	// Parse derivation paths
	derivPaths, err := worker.ParseDerivationPaths(cfg.derivationPaths)
	if err != nil {
		log.Fatalf("Invalid derivation paths: %v", err)
	}

	log.Printf("Derivation paths: %v", worker.FormatDerivationPaths(derivPaths))
	log.Printf("Address indexes: 0-%d", cfg.addressIndexes-1)
	log.Printf("Total addresses per mnemonic: %d", len(derivPaths)*cfg.addressIndexes)

	// Create chits worker configuration
	chitsCfg := worker.ChitsConfig{
		Chits:           chitsData,
		AddressIndexes:  cfg.addressIndexes,
		DerivationPaths: derivPaths,
		Verbose:         cfg.verbose,
		CheckpointPath:  cfg.checkpointPath,
		CheckpointEvery: cfg.checkpointEvery,
	}

	// For chits mode, we typically use a single worker since the permutation
	// space is already defined by the chits
	log.Println("Starting chits recovery worker...")
	w := worker.NewChitsWorker(hashSet, chitsCfg)

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer w.Close()

		workerMatches := w.Run(ctx)
		for match := range workerMatches {
			matchChan <- match
		}
	}()

	// Stats function
	statsFn = func() (int64, int64) {
		stats := w.Stats()
		return stats.AddressesChecked, stats.MnemonicsGenerated
	}

	waitFn = func() {
		wg.Wait()
		close(matchChan)
	}

	return
}

// parseDerivationPaths parses comma-separated BIP purpose numbers
func parseDerivationPaths(pathStr string) ([]uint32, error) {
	if pathStr == "" {
		return []uint32{44, 49, 84, 86}, nil // Default
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

// formatDerivationPaths formats purposes for display
func formatDerivationPaths(paths []uint32) string {
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
