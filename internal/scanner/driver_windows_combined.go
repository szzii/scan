//go:build windows

package scanner

import (
	"context"
	"fmt"

	"github.com/scanserver/scanner-service/pkg/models"
)

// CombinedWindowsDriver combines WIA and TWAIN support
type CombinedWindowsDriver struct {
	wiaDriver   *WindowsDriver
	twainDriver *TWAINDriver
	useWIA      bool
}

func newCombinedDriver() (ScannerDriver, error) {
	driver := &CombinedWindowsDriver{}
	var wiaErr, twainErr error

	// Initialize BOTH drivers to detect all scanners
	// Some scanners (like D2800+) may only be visible via TWAIN
	// while others are only visible via WIA

	fmt.Println("Initializing combined WIA+TWAIN driver...")

	// Try WIA (modern Windows scanners)
	wiaDriver, err := newWIADriver()
	if err == nil {
		driver.wiaDriver = wiaDriver
		driver.useWIA = true
		fmt.Println("✓ WIA driver initialized successfully")
	} else {
		wiaErr = err
		fmt.Printf("⚠ WIA driver initialization failed: %v\n", err)
	}

	// Also try TWAIN (legacy scanners and some USB devices)
	twainDriver, err := newTWAINDriver()
	if err == nil {
		driver.twainDriver = twainDriver
		fmt.Println("✓ TWAIN driver initialized successfully")
	} else {
		twainErr = err
		fmt.Printf("⚠ TWAIN driver initialization failed: %v\n", err)
	}

	// Require at least one driver to work
	if driver.wiaDriver == nil && driver.twainDriver == nil {
		return nil, fmt.Errorf("no scanner driver available. WIA error: %v, TWAIN error: %v", wiaErr, twainErr)
	}

	fmt.Printf("Combined driver initialized with WIA=%v, TWAIN=%v\n", driver.wiaDriver != nil, driver.twainDriver != nil)
	return driver, nil
}

func (d *CombinedWindowsDriver) ListScanners(ctx context.Context) ([]models.Scanner, error) {
	var allScanners []models.Scanner

	fmt.Println("\n=== Combined Driver: Enumerating Scanners ===")

	// Try to get scanners from both drivers
	if d.wiaDriver != nil {
		fmt.Println("Checking WIA driver for scanners...")
		scanners, err := d.wiaDriver.ListScanners(ctx)
		if err == nil {
			fmt.Printf("✓ WIA found %d scanner(s)\n", len(scanners))
			for _, scanner := range scanners {
				scanner.ID = "wia:" + scanner.ID // Prefix to identify source
				allScanners = append(allScanners, scanner)
			}
		} else {
			fmt.Printf("⚠ WIA scanner enumeration failed: %v\n", err)
		}
	}

	if d.twainDriver != nil {
		fmt.Println("\nChecking TWAIN driver for scanners...")
		scanners, err := d.twainDriver.ListScanners(ctx)
		if err == nil {
			fmt.Printf("✓ TWAIN found %d scanner(s)\n", len(scanners))
			for _, scanner := range scanners {
				scanner.ID = "twain:" + scanner.ID // Prefix to identify source
				allScanners = append(allScanners, scanner)
			}
		} else {
			fmt.Printf("⚠ TWAIN scanner enumeration failed: %v\n", err)
		}
	}

	fmt.Printf("\n=== Total: %d scanner(s) found ===\n\n", len(allScanners))

	if len(allScanners) == 0 {
		return nil, fmt.Errorf("no scanners found via WIA or TWAIN")
	}

	return allScanners, nil
}

func (d *CombinedWindowsDriver) GetScanner(ctx context.Context, scannerID string) (*models.Scanner, error) {
	// Remove protocol prefix to get actual scanner ID
	actualID := scannerID
	if len(scannerID) > 4 {
		if scannerID[:4] == "wia:" {
			actualID = scannerID[4:]
			if d.wiaDriver != nil {
				return d.wiaDriver.GetScanner(ctx, actualID)
			}
		} else if scannerID[:6] == "twain:" {
			actualID = scannerID[6:]
			if d.twainDriver != nil {
				return d.twainDriver.GetScanner(ctx, actualID)
			}
		}
	}

	// Fallback: try both drivers
	if d.wiaDriver != nil {
		scanner, err := d.wiaDriver.GetScanner(ctx, actualID)
		if err == nil {
			return scanner, nil
		}
	}

	if d.twainDriver != nil {
		return d.twainDriver.GetScanner(ctx, actualID)
	}

	return nil, fmt.Errorf("scanner not found: %s", scannerID)
}

func (d *CombinedWindowsDriver) Scan(ctx context.Context, scannerID string, params models.ScanParams, progressCallback func(int)) ([]models.ScanResult, error) {
	// Remove protocol prefix to get actual scanner ID
	actualID := scannerID
	if len(scannerID) > 4 {
		if scannerID[:4] == "wia:" {
			actualID = scannerID[4:]
			if d.wiaDriver != nil {
				return d.wiaDriver.Scan(ctx, actualID, params, progressCallback)
			}
		} else if scannerID[:6] == "twain:" {
			actualID = scannerID[6:]
			if d.twainDriver != nil {
				return d.twainDriver.Scan(ctx, actualID, params, progressCallback)
			}
		}
	}

	// Fallback: try to determine which driver to use based on scanner ID
	if d.wiaDriver != nil {
		_, err := d.wiaDriver.GetScanner(ctx, actualID)
		if err == nil {
			return d.wiaDriver.Scan(ctx, actualID, params, progressCallback)
		}
	}

	if d.twainDriver != nil {
		return d.twainDriver.Scan(ctx, actualID, params, progressCallback)
	}

	return nil, fmt.Errorf("no driver available for scanner: %s", scannerID)
}

func (d *CombinedWindowsDriver) CancelScan(ctx context.Context, scannerID string) error {
	// Remove protocol prefix to get actual scanner ID
	actualID := scannerID
	if len(scannerID) > 4 {
		if scannerID[:4] == "wia:" {
			actualID = scannerID[4:]
			if d.wiaDriver != nil {
				return d.wiaDriver.CancelScan(ctx, actualID)
			}
		} else if scannerID[:6] == "twain:" {
			actualID = scannerID[6:]
			if d.twainDriver != nil {
				return d.twainDriver.CancelScan(ctx, actualID)
			}
		}
	}

	// Fallback: try both drivers
	if d.wiaDriver != nil {
		err := d.wiaDriver.CancelScan(ctx, actualID)
		if err == nil {
			return nil
		}
	}

	if d.twainDriver != nil {
		return d.twainDriver.CancelScan(ctx, actualID)
	}

	return fmt.Errorf("scanner not found: %s", scannerID)
}

func (d *CombinedWindowsDriver) WatchLidStatus(ctx context.Context, scannerID string, callback func(lidClosed bool)) error {
	// Remove protocol prefix to get actual scanner ID
	actualID := scannerID
	if len(scannerID) > 4 {
		if scannerID[:4] == "wia:" {
			actualID = scannerID[4:]
			if d.wiaDriver != nil {
				return d.wiaDriver.WatchLidStatus(ctx, actualID, callback)
			}
		} else if scannerID[:6] == "twain:" {
			actualID = scannerID[6:]
			if d.twainDriver != nil {
				return d.twainDriver.WatchLidStatus(ctx, actualID, callback)
			}
		}
	}

	// Fallback: try WIA first
	if d.wiaDriver != nil {
		return d.wiaDriver.WatchLidStatus(ctx, actualID, callback)
	}

	if d.twainDriver != nil {
		return d.twainDriver.WatchLidStatus(ctx, actualID, callback)
	}

	return fmt.Errorf("no driver available")
}

func (d *CombinedWindowsDriver) Close() error {
	if d.wiaDriver != nil {
		d.wiaDriver.Close()
	}
	if d.twainDriver != nil {
		d.twainDriver.Close()
	}
	return nil
}

// Helper function to create WIA driver
func newWIADriver() (*WindowsDriver, error) {
	return newWIADriverInternal()
}
