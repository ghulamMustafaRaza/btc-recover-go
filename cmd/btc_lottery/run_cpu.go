//go:build !cuda

package main

import (
	"context"
	"log"
	"sync"

	"btc_recover/internal/chits"
	"btc_recover/internal/lookup"
	"btc_recover/internal/worker"
)

// runWorkers starts CPU workers (non-GPU build).
func runWorkers(ctx context.Context, hashSet *lookup.AddressHashSet, cfg workerConfig) (matchChan chan worker.Match, statsFn func() (int64, int64), waitFn func()) {
	if cfg.useGPU {
		log.Println("WARNING: GPU acceleration requested but not compiled with -tags cuda")
		log.Println("Falling back to CPU-only mode")
	}

	matchChan = make(chan worker.Match, 100)
	var wg sync.WaitGroup

	// Check if chits mode is enabled
	if cfg.chitsMode {
		return runChitsWorkers(ctx, hashSet, cfg)
	}

	workerCfg := worker.Config{
		AddressIndexes: cfg.addressIndexes,
		EntropyBits:    cfg.entropyBits,
		GPUBatchSize:   cfg.gpuBatchSize,
		UseGPU:         false,
		Verbose:        cfg.verbose,
	}

	// Keep references to workers for stats
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

	// Stats function queries all workers in real-time
	statsFn = func() (int64, int64) {
		var totalAddr, totalMnem int64
		for _, w := range workers {
			stats := w.Stats()
			totalAddr += stats.AddressesChecked
			totalMnem += stats.MnemonicsGenerated
		}
		return totalAddr, totalMnem
	}

	waitFn = func() {
		wg.Wait()
		close(matchChan)
	}

	return
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
