//go:build linux

package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/scanserver/scanner-service/pkg/models"
)

// LinuxDriver implements ScannerDriver for Linux using SANE (Scanner Access Now Easy)
type LinuxDriver struct {
	scanners map[string]*models.Scanner
}

func newPlatformDriver() (ScannerDriver, error) {
	// In a real implementation, initialize SANE library
	// sane_init()
	return &LinuxDriver{
		scanners: make(map[string]*models.Scanner),
	}, nil
}

func (d *LinuxDriver) ListScanners(ctx context.Context) ([]models.Scanner, error) {
	// In a real implementation, this would call sane_get_devices()
	// and parse the device list

	scanner := models.Scanner{
		ID:           "scanner-001",
		Name:         "HP LaserJet Scanner",
		Model:        "HP LaserJet Pro MFP M428fdw",
		Manufacturer: "HP",
		Status:       "idle",
		Capabilities: models.Capability{
			MaxWidth:        8500, // A4 width in 0.01mm
			MaxHeight:       11700, // A4 height in 0.01mm
			Resolutions:     []int{100, 150, 200, 300, 600, 1200},
			ColorModes:      []string{"Color", "Grayscale", "BlackAndWhite"},
			DocumentFormats: []string{"PDF", "JPEG", "PNG", "TIFF"},
			FeederEnabled:   true,
			DuplexEnabled:   true,
		},
		LastSeen: time.Now(),
	}

	d.scanners[scanner.ID] = &scanner
	return []models.Scanner{scanner}, nil
}

func (d *LinuxDriver) GetScanner(ctx context.Context, scannerID string) (*models.Scanner, error) {
	scanner, ok := d.scanners[scannerID]
	if !ok {
		return nil, fmt.Errorf("scanner not found: %s", scannerID)
	}
	return scanner, nil
}

func (d *LinuxDriver) Scan(ctx context.Context, scannerID string, params models.ScanParams, progressCallback func(int)) ([]models.ScanResult, error) {
	scanner, err := d.GetScanner(ctx, scannerID)
	if err != nil {
		return nil, err
	}

	if scanner.Status != "idle" {
		return nil, fmt.Errorf("scanner is busy")
	}

	scanner.Status = "scanning"
	defer func() {
		scanner.Status = "idle"
	}()

	// In a real implementation, this would:
	// 1. Open SANE device: sane_open()
	// 2. Set scan parameters: sane_control_option()
	// 3. Start scan: sane_start()
	// 4. Read scan data: sane_read()
	// 5. Save to file format
	// 6. Close device: sane_close()

	var results []models.ScanResult
	pageCount := params.PageCount
	if pageCount == 0 {
		pageCount = 1
	}

	for i := 0; i < pageCount; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		progress := (i * 100) / pageCount
		if progressCallback != nil {
			progressCallback(progress)
		}

		// Simulate scan time
		time.Sleep(2 * time.Second)

		result := models.ScanResult{
			PageNumber: i + 1,
			FilePath:   fmt.Sprintf("scan_%s_page_%d.%s", time.Now().Format("20060102_150405"), i+1, params.Format),
			FileSize:   1024 * 1024,
			Format:     params.Format,
			Width:      params.Width,
			Height:     params.Height,
		}

		results = append(results, result)
	}

	if progressCallback != nil {
		progressCallback(100)
	}

	return results, nil
}

func (d *LinuxDriver) CancelScan(ctx context.Context, scannerID string) error {
	scanner, err := d.GetScanner(ctx, scannerID)
	if err != nil {
		return err
	}

	// Call sane_cancel() in real implementation
	scanner.Status = "idle"
	return nil
}

func (d *LinuxDriver) WatchLidStatus(ctx context.Context, scannerID string, callback func(lidClosed bool)) error {
	// SANE doesn't provide direct lid status events
	// Could monitor using inotify on device files or poll device status
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Poll scanner status or use udev events
			}
		}
	}()

	return nil
}

func (d *LinuxDriver) Close() error {
	// Call sane_exit() in real implementation
	return nil
}
