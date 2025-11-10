package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scanserver/scanner-service/pkg/models"
)

// BatchScanPerformer performs batch scanning operations
// Implements NAPS2's batch scanning workflow (BatchScanPerformer.cs)
type BatchScanPerformer struct {
	driver ScannerDriver
}

// NewBatchScanPerformer creates a new batch scan performer
func NewBatchScanPerformer(driver ScannerDriver) *BatchScanPerformer {
	return &BatchScanPerformer{
		driver: driver,
	}
}

// PerformBatchScan executes a batch scan according to settings
// Implements NAPS2's PerformBatchScan method (BatchScanPerformer.cs:36-42)
func (b *BatchScanPerformer) PerformBatchScan(
	ctx context.Context,
	scannerID string,
	settings models.BatchSettings,
	progressCallback func(models.BatchScanProgress),
) ([][]models.ScanResult, error) {
	state := &batchState{
		driver:           b.driver,
		scannerID:        scannerID,
		settings:         settings,
		progressCallback: progressCallback,
		scans:            make([][]models.ScanResult, 0),
		ctx:              ctx,
	}

	return state.do()
}

// batchState manages the state of a batch scan operation
// Implements NAPS2's BatchState inner class (BatchScanPerformer.cs:44-308)
type batchState struct {
	driver           ScannerDriver
	scannerID        string
	settings         models.BatchSettings
	progressCallback func(models.BatchScanProgress)
	scans            [][]models.ScanResult
	ctx              context.Context
}

// do executes the batch scan workflow
// Implements NAPS2's Do method (BatchScanPerformer.cs:99-126)
func (s *batchState) do() ([][]models.ScanResult, error) {
	// Input phase: perform scans
	if err := s.input(); err != nil {
		// Try to save what we have even if input failed
		if saveErr := s.output(); saveErr != nil {
			return s.scans, fmt.Errorf("input failed: %w, output failed: %v", err, saveErr)
		}
		return s.scans, err
	}

	// Output phase: save results
	if err := s.output(); err != nil {
		return s.scans, err
	}

	return s.scans, nil
}

// input performs the scanning phase
// Implements NAPS2's Input method (BatchScanPerformer.cs:128-168)
func (s *batchState) input() error {
	switch s.settings.ScanType {
	case models.BatchScanSingle:
		// Single scan
		return s.inputOneScan(-1)

	case models.BatchScanMultipleWithDelay:
		// Multiple scans with delay
		for i := 0; i < s.settings.ScanCount; i++ {
			s.sendProgress("scanning", i+1, s.settings.ScanCount, 0, 0,
				fmt.Sprintf("Waiting for scan %d of %d", i+1, s.settings.ScanCount))

			// Wait between scans (except first scan)
			if i != 0 {
				select {
				case <-time.After(time.Duration(s.settings.ScanIntervalSeconds * float64(time.Second))):
					// Continue
				case <-s.ctx.Done():
					return s.ctx.Err()
				}
			}

			if err := s.inputOneScan(i); err != nil {
				return err
			}
		}
		return nil

	case models.BatchScanMultipleWithPrompt:
		// Multiple scans with user prompt
		// For API use, this is equivalent to single scan
		// In a GUI, this would prompt the user after each scan
		i := 0
		for {
			s.sendProgress("scanning", i+1, -1, 0, 0,
				fmt.Sprintf("Scanning batch %d", i+1))

			if err := s.inputOneScan(i); err != nil {
				return err
			}

			i++

			// For API, we don't have user prompt capability
			// So we treat this as single scan mode
			break
		}
		return nil

	default:
		return fmt.Errorf("unknown batch scan type: %s", s.settings.ScanType)
	}
}

// inputOneScan performs a single scan operation
// Implements NAPS2's InputOneScan method (BatchScanPerformer.cs:175-199)
func (s *batchState) inputOneScan(scanNumber int) error {
	scan := make([]models.ScanResult, 0)

	// Progress callback
	pageNumber := 1
	if scanNumber == -1 {
		s.sendProgress("scanning", 1, 1, pageNumber, 0,
			fmt.Sprintf("Scanning page %d", pageNumber))
	} else {
		s.sendProgress("scanning", scanNumber+1, s.settings.ScanCount, pageNumber, 0,
			fmt.Sprintf("Scanning page %d of scan %d", pageNumber, scanNumber+1))
	}

	// Perform the scan
	results, err := s.driver.Scan(s.ctx, s.scannerID, s.settings.ScanParams, func(progress int) {
		pageNumber++
		if scanNumber == -1 {
			s.sendProgress("scanning", 1, 1, pageNumber, 0,
				fmt.Sprintf("Scanning page %d", pageNumber))
		} else {
			s.sendProgress("scanning", scanNumber+1, s.settings.ScanCount, pageNumber, 0,
				fmt.Sprintf("Scanning page %d of scan %d", pageNumber, scanNumber+1))
		}
	})

	if err != nil {
		// If we got some results before the error, save them
		if len(scan) > 0 {
			s.scans = append(s.scans, scan)
		}
		return err
	}

	if len(results) == 0 {
		// No results, possibly cancelled
		return fmt.Errorf("no pages scanned")
	}

	s.scans = append(s.scans, results)
	return nil
}

// output saves the scan results
// Implements NAPS2's Output method (BatchScanPerformer.cs:227-261)
func (s *batchState) output() error {
	s.sendProgress("saving", 0, 0, 0, len(s.scans), "Saving scan results...")

	// Collect all images from all scans
	allImages := make([]models.ScanResult, 0)
	for _, scan := range s.scans {
		allImages = append(allImages, scan...)
	}

	switch s.settings.OutputType {
	case models.BatchOutputLoad:
		// Load mode: just return the results, no saving
		return nil

	case models.BatchOutputSingleFile:
		// Single file: save all pages as one file
		if len(allImages) > 0 {
			return s.save(0, allImages)
		}
		return nil

	case models.BatchOutputMultipleFiles:
		// Multiple files: separate based on SaveSeparator
		switch s.settings.SaveSeparator {
		case models.SaveSeparatorFilePerScan:
			// One file per scan
			for i, scan := range s.scans {
				if err := s.save(i, scan); err != nil {
					return err
				}
			}

		case models.SaveSeparatorFilePerPage:
			// One file per page
			for i, image := range allImages {
				if err := s.save(i, []models.ScanResult{image}); err != nil {
					return err
				}
			}

		default:
			// Default: one file per scan
			for i, scan := range s.scans {
				if err := s.save(i, scan); err != nil {
					return err
				}
			}
		}
		return nil

	default:
		return fmt.Errorf("unknown output type: %s", s.settings.OutputType)
	}
}

// save saves a set of images to a file
// Implements NAPS2's Save method (BatchScanPerformer.cs:263-298)
func (s *batchState) save(index int, images []models.ScanResult) error {
	if len(images) == 0 {
		return nil
	}

	// Substitute placeholders in save path
	savePath := s.substitutePlaceholders(s.settings.SavePath, index)

	// Ensure directory exists
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(savePath))

	if ext == ".pdf" {
		// PDF export would go here
		// For now, we'll just copy the files to the target location
		// In a full implementation, you would merge all images into a PDF
		return fmt.Errorf("PDF export not yet implemented")
	}

	// For image files, handle based on number of images
	if len(images) == 1 {
		// Single image: copy to target location
		return s.copyFile(images[0].FilePath, savePath)
	}

	// Multiple images: save each with index suffix
	for i, image := range images {
		indexedPath := s.addIndexToPath(savePath, i)
		if err := s.copyFile(image.FilePath, indexedPath); err != nil {
			return err
		}
	}

	return nil
}

// substitutePlaceholders replaces placeholders in the save path
// Simplified version of NAPS2's Placeholders.Substitute
func (s *batchState) substitutePlaceholders(path string, index int) string {
	now := time.Now()

	replacements := map[string]string{
		"$(n)":    fmt.Sprintf("%d", index+1),        // Sequential number
		"$(yyyy)": now.Format("2006"),                // Year
		"$(yy)":   now.Format("06"),                  // Year (2-digit)
		"$(MM)":   now.Format("01"),                  // Month
		"$(dd)":   now.Format("02"),                  // Day
		"$(hh)":   now.Format("15"),                  // Hour
		"$(mm)":   now.Format("04"),                  // Minute
		"$(ss)":   now.Format("05"),                  // Second
	}

	result := path
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// addIndexToPath adds an index to a file path before the extension
func (s *batchState) addIndexToPath(path string, index int) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return fmt.Sprintf("%s_%d%s", base, index+1, ext)
}

// copyFile copies a file from src to dst
func (s *batchState) copyFile(src, dst string) error {
	// Read source file
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(dst, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %w", err)
	}

	return nil
}

// sendProgress sends a progress update via callback
func (s *batchState) sendProgress(stage string, currentScan, totalScans, currentPage, totalPages int, message string) {
	if s.progressCallback == nil {
		return
	}

	// Calculate percentage
	percentComplete := 0
	if stage == "scanning" {
		if totalScans > 0 {
			percentComplete = (currentScan * 50) / totalScans
		} else {
			percentComplete = 25
		}
	} else if stage == "saving" {
		percentComplete = 50 + 50 // 100% when saving
	}

	s.progressCallback(models.BatchScanProgress{
		Stage:           stage,
		CurrentScan:     currentScan,
		TotalScans:      totalScans,
		CurrentPage:     currentPage,
		TotalPages:      totalPages,
		Message:         message,
		PercentComplete: percentComplete,
	})
}
