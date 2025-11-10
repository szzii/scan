package scanner

import (
	"context"
	"log"
	"time"

	"github.com/scanserver/scanner-service/internal/config"
	"github.com/scanserver/scanner-service/pkg/models"
)

// AutoScanManager manages automatic scanning on lid close
type AutoScanManager struct {
	manager       *Manager
	config        *config.AutoScanConfig
	scanCallback  func(*models.ScanJob)
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewAutoScanManager creates a new auto-scan manager
func NewAutoScanManager(manager *Manager, cfg *config.AutoScanConfig, scanCallback func(*models.ScanJob)) *AutoScanManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &AutoScanManager{
		manager:      manager,
		config:       cfg,
		scanCallback: scanCallback,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts monitoring for lid close events
func (a *AutoScanManager) Start() error {
	if !a.config.Enabled {
		log.Println("Auto-scan is disabled")
		return nil
	}

	// Get scanners to monitor
	scanners, err := a.manager.ListScanners(a.ctx)
	if err != nil {
		return err
	}

	if len(scanners) == 0 {
		log.Println("No scanners available for auto-scan")
		return nil
	}

	// Find target scanner
	var targetScanner *models.Scanner
	if a.config.ScannerID != "" {
		for i := range scanners {
			if scanners[i].ID == a.config.ScannerID {
				targetScanner = &scanners[i]
				break
			}
		}
		if targetScanner == nil {
			log.Printf("Specified scanner %s not found, using first available", a.config.ScannerID)
		}
	}

	if targetScanner == nil {
		targetScanner = &scanners[0]
	}

	log.Printf("Starting auto-scan monitoring for scanner: %s (%s)", targetScanner.Name, targetScanner.ID)

	// Start watching lid status
	err = a.manager.WatchLidStatus(a.ctx, targetScanner.ID, func(lidClosed bool) {
		if lidClosed {
			a.handleLidClosed(targetScanner.ID)
		}
	})

	if err != nil {
		return err
	}

	return nil
}

// handleLidClosed handles lid close event
func (a *AutoScanManager) handleLidClosed(scannerID string) {
	log.Printf("Lid closed detected on scanner %s, waiting %d seconds before scanning...",
		scannerID, a.config.LidCloseDelay)

	// Wait for configured delay
	time.Sleep(time.Duration(a.config.LidCloseDelay) * time.Second)

	// Create scan job
	job := &models.ScanJob{
		ID:        models.GenerateUUID(),
		ScannerID: scannerID,
		Status:    "pending",
		Progress:  0,
		Parameters: models.ScanParams{
			Resolution: a.config.DefaultParams.Resolution,
			ColorMode:  a.config.DefaultParams.ColorMode,
			Format:     a.config.DefaultParams.Format,
			Width:      210, // A4 width in mm
			Height:     297, // A4 height in mm
			Brightness: 0,
			Contrast:   0,
			UseDuplex:  a.config.DefaultParams.UseDuplex,
			UseFeeder:  a.config.DefaultParams.UseFeeder,
			PageCount:  1,
		},
		Results:   []models.ScanResult{},
		CreatedAt: time.Now(),
	}

	log.Printf("Starting auto-scan job %s", job.ID)

	// Execute scan
	go a.executeScan(job)
}

// executeScan executes the scan job
func (a *AutoScanManager) executeScan(job *models.ScanJob) {
	job.Status = "processing"

	// Notify callback
	if a.scanCallback != nil {
		a.scanCallback(job)
	}

	// Progress callback
	progressCallback := func(progress int) {
		job.Progress = progress
		log.Printf("Auto-scan job %s progress: %d%%", job.ID, progress)

		if a.scanCallback != nil {
			a.scanCallback(job)
		}
	}

	// Execute scan
	results, err := a.manager.Scan(a.ctx, job.ScannerID, job.Parameters, progressCallback)

	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		log.Printf("Auto-scan job %s failed: %v", job.ID, err)
	} else {
		job.Status = "completed"
		job.Results = results
		job.Progress = 100
		log.Printf("Auto-scan job %s completed successfully, %d pages scanned", job.ID, len(results))
	}

	now := time.Now()
	job.CompletedAt = &now

	// Final callback
	if a.scanCallback != nil {
		a.scanCallback(job)
	}
}

// Stop stops auto-scan monitoring
func (a *AutoScanManager) Stop() {
	log.Println("Stopping auto-scan manager")
	a.cancel()
}
