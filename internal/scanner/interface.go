package scanner

import (
	"context"

	"github.com/scanserver/scanner-service/pkg/models"
)

// ScannerDriver is the interface for platform-specific scanner implementations
type ScannerDriver interface {
	// ListScanners returns all available scanners
	ListScanners(ctx context.Context) ([]models.Scanner, error)

	// GetScanner returns a specific scanner by ID
	GetScanner(ctx context.Context, scannerID string) (*models.Scanner, error)

	// Scan performs a scan operation
	Scan(ctx context.Context, scannerID string, params models.ScanParams, progressCallback func(int)) ([]models.ScanResult, error)

	// CancelScan cancels an ongoing scan operation
	CancelScan(ctx context.Context, scannerID string) error

	// WatchLidStatus watches for scanner lid close events
	WatchLidStatus(ctx context.Context, scannerID string, callback func(lidClosed bool)) error

	// Close releases resources
	Close() error
}

// Manager manages scanner operations across platforms
type Manager struct {
	driver ScannerDriver
}

// NewManager creates a new scanner manager
func NewManager() (*Manager, error) {
	driver, err := newPlatformDriver()
	if err != nil {
		return nil, err
	}

	return &Manager{
		driver: driver,
	}, nil
}

// ListScanners returns all available scanners
func (m *Manager) ListScanners(ctx context.Context) ([]models.Scanner, error) {
	return m.driver.ListScanners(ctx)
}

// GetScanner returns a specific scanner
func (m *Manager) GetScanner(ctx context.Context, scannerID string) (*models.Scanner, error) {
	return m.driver.GetScanner(ctx, scannerID)
}

// Scan performs a scan operation
func (m *Manager) Scan(ctx context.Context, scannerID string, params models.ScanParams, progressCallback func(int)) ([]models.ScanResult, error) {
	return m.driver.Scan(ctx, scannerID, params, progressCallback)
}

// CancelScan cancels an ongoing scan
func (m *Manager) CancelScan(ctx context.Context, scannerID string) error {
	return m.driver.CancelScan(ctx, scannerID)
}

// WatchLidStatus watches for lid close events
func (m *Manager) WatchLidStatus(ctx context.Context, scannerID string, callback func(lidClosed bool)) error {
	return m.driver.WatchLidStatus(ctx, scannerID, callback)
}

// Close releases resources
func (m *Manager) Close() error {
	if m.driver != nil {
		return m.driver.Close()
	}
	return nil
}

// GetDriver returns the underlying scanner driver
// Used by batch scan performer to access low-level driver functions
func (m *Manager) GetDriver() ScannerDriver {
	return m.driver
}
