//go:build windows

package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/scanserver/scanner-service/pkg/models"
)

// TWAIN Constants
const (
	TWAIN_DLL = "TWAINDSM.dll"

	// TWAIN message codes
	MSG_OPENDSM     = 0x0301
	MSG_CLOSEDSM    = 0x0302
	MSG_OPENDS      = 0x0401
	MSG_CLOSEDS     = 0x0402
	MSG_USERSELECT  = 0x0403
	MSG_GET         = 0x0001
	MSG_GETCURRENT  = 0x0002
	MSG_GETDEFAULT  = 0x0003
	MSG_GETFIRST    = 0x0004  // Get first item in enumeration
	MSG_GETNEXT     = 0x0005  // Get next item in enumeration
	MSG_SET         = 0x0006
	MSG_ENABLEDS    = 0x1001
	MSG_ENABLEDSUIONLY = 0x1002
	MSG_DISABLEDS   = 0x1201
	MSG_XFERREADY   = 0x0101
	MSG_ENDXFER     = 0x0701

	// Data Group
	DG_CONTROL = 0x0001
	DG_IMAGE   = 0x0002

	// Data Argument Type
	DAT_IDENTITY   = 0x0003
	DAT_USERINTERFACE = 0x0009
	DAT_STATUS     = 0x0008
	DAT_CAPABILITY = 0x0001
	DAT_IMAGEINFO  = 0x0101
	DAT_IMAGENATIVEXFER = 0x0104

	// Capability constants
	CAP_XFERCOUNT  = 0x0001
	ICAP_XRESOLUTION = 0x1118
	ICAP_YRESOLUTION = 0x1119
	ICAP_PIXELTYPE = 0x0101

	// Pixel types
	TWPT_BW    = 0
	TWPT_GRAY  = 1
	TWPT_RGB   = 2

	// Return codes
	TWRC_SUCCESS    = 0
	TWRC_FAILURE    = 1
	TWRC_CHECKSTATUS = 2
	TWRC_CANCEL     = 3
	TWRC_DSEVENT    = 4
	TWRC_NOTDSEVENT = 5
	TWRC_XFERDONE   = 6
	TWRC_ENDOFLIST  = 7  // No more items in enumeration
)

// TWAIN structures
type TW_IDENTITY struct {
	Id              uint32
	Version         [8]uint16
	ProtocolMajor   uint16
	ProtocolMinor   uint16
	SupportedGroups uint32
	Manufacturer    [34]uint16
	ProductFamily   [34]uint16
	ProductName     [34]uint16
}

type TW_USERINTERFACE struct {
	ShowUI   uint16
	ModalUI  uint16
	hParent  uintptr
}

// TWAINDriver implements TWAIN protocol support
type TWAINDriver struct {
	scanners    map[string]*models.Scanner
	dsmLib      *syscall.DLL
	dsmEntry    *syscall.Proc
	appIdentity *TW_IDENTITY
	dsIdentity  *TW_IDENTITY
	initialized bool
}

func newTWAINDriver() (*TWAINDriver, error) {
	driver := &TWAINDriver{
		scanners: make(map[string]*models.Scanner),
	}

	// Try to load TWAIN DSM
	dsmLib, err := syscall.LoadDLL(TWAIN_DLL)
	if err != nil {
		return nil, fmt.Errorf("TWAIN DSM not found (TWAINDSM.dll). Please install TWAIN drivers: %w", err)
	}

	dsmEntry, err := dsmLib.FindProc("DSM_Entry")
	if err != nil {
		return nil, fmt.Errorf("failed to find DSM_Entry: %w", err)
	}

	driver.dsmLib = dsmLib
	driver.dsmEntry = dsmEntry

	// Initialize application identity
	driver.appIdentity = &TW_IDENTITY{
		ProtocolMajor:   2,
		ProtocolMinor:   3,
		SupportedGroups: DG_CONTROL | DG_IMAGE,
	}
	copy(driver.appIdentity.Manufacturer[:], utf16FromString("Scanner Service"))
	copy(driver.appIdentity.ProductFamily[:], utf16FromString("Document Scanner"))
	copy(driver.appIdentity.ProductName[:], utf16FromString("ScanServer"))

	driver.initialized = true
	return driver, nil
}

func utf16FromString(s string) []uint16 {
	result := make([]uint16, len(s))
	for i, c := range s {
		result[i] = uint16(c)
	}
	return result
}

func (d *TWAINDriver) ListScanners(ctx context.Context) ([]models.Scanner, error) {
	if !d.initialized {
		return nil, fmt.Errorf("TWAIN driver not initialized")
	}

	fmt.Println("TWAIN: Starting scanner enumeration...")

	// Step 1: Open DSM (Data Source Manager)
	fmt.Println("TWAIN: Opening Data Source Manager...")
	ret, _, _ := d.dsmEntry.Call(
		uintptr(unsafe.Pointer(d.appIdentity)),
		0,
		DG_CONTROL,
		DAT_IDENTITY,
		MSG_OPENDSM,
		0,
	)

	if ret != TWRC_SUCCESS {
		return nil, fmt.Errorf("TWAIN: failed to open DSM, error code: %d", ret)
	}
	fmt.Println("TWAIN: DSM opened successfully")

	// Ensure we close DSM when done
	defer func() {
		d.dsmEntry.Call(
			uintptr(unsafe.Pointer(d.appIdentity)),
			0,
			DG_CONTROL,
			DAT_IDENTITY,
			MSG_CLOSEDSM,
			0,
		)
		fmt.Println("TWAIN: DSM closed")
	}()

	var scanners []models.Scanner

	// Step 2: Get first data source
	fmt.Println("TWAIN: Enumerating data sources...")
	var dsIdentity TW_IDENTITY
	ret, _, _ = d.dsmEntry.Call(
		uintptr(unsafe.Pointer(d.appIdentity)),
		0,
		DG_CONTROL,
		DAT_IDENTITY,
		MSG_GETFIRST,
		uintptr(unsafe.Pointer(&dsIdentity)),
	)

	if ret == TWRC_ENDOFLIST {
		fmt.Println("TWAIN: No data sources found")
		return nil, fmt.Errorf("no TWAIN data sources found")
	}

	if ret != TWRC_SUCCESS {
		fmt.Printf("TWAIN: Failed to get first data source, error code: %d\n", ret)
		return nil, fmt.Errorf("TWAIN: failed to enumerate data sources")
	}

	// Process first data source
	scannerCount := 0
	for {
		scannerCount++

		// Convert UTF-16 strings to Go strings
		productName := utf16ToString(dsIdentity.ProductName[:])
		manufacturer := utf16ToString(dsIdentity.Manufacturer[:])
		productFamily := utf16ToString(dsIdentity.ProductFamily[:])

		fmt.Printf("TWAIN: Found data source %d: %s (%s)\n", scannerCount, productName, manufacturer)

		// Create scanner object
		scanner := models.Scanner{
			ID:           fmt.Sprintf("twain:%d", dsIdentity.Id),
			Name:         productName,
			Model:        productFamily,
			Manufacturer: manufacturer,
			Status:       "idle",
			Capabilities: models.Capability{
				MaxWidth:        2100,
				MaxHeight:       2970,
				Resolutions:     []int{75, 100, 150, 200, 300, 600, 1200},
				ColorModes:      []string{"Color", "Grayscale", "BlackAndWhite"},
				DocumentFormats: []string{"JPEG", "PNG", "TIFF", "BMP"},
				FeederEnabled:   true,
				DuplexEnabled:   false,
			},
			LastSeen: time.Now(),
		}

		d.scanners[scanner.ID] = &scanner
		scanners = append(scanners, scanner)

		// Step 3: Get next data source
		ret, _, _ = d.dsmEntry.Call(
			uintptr(unsafe.Pointer(d.appIdentity)),
			0,
			DG_CONTROL,
			DAT_IDENTITY,
			MSG_GETNEXT,
			uintptr(unsafe.Pointer(&dsIdentity)),
		)

		if ret == TWRC_ENDOFLIST {
			// Normal end of enumeration
			break
		}

		if ret != TWRC_SUCCESS {
			// Error, but we got at least some scanners
			fmt.Printf("TWAIN: Enumeration ended with error code: %d\n", ret)
			break
		}
	}

	fmt.Printf("TWAIN: Enumeration complete. Found %d TWAIN data source(s)\n", len(scanners))

	if len(scanners) == 0 {
		return nil, fmt.Errorf("no TWAIN data sources found")
	}

	return scanners, nil
}

// utf16ToString converts UTF-16 array to Go string
func utf16ToString(u16 []uint16) string {
	// Find null terminator
	length := 0
	for i, c := range u16 {
		if c == 0 {
			length = i
			break
		}
	}
	if length == 0 {
		length = len(u16)
	}

	// Convert to runes
	runes := make([]rune, length)
	for i := 0; i < length; i++ {
		runes[i] = rune(u16[i])
	}
	return string(runes)
}

func (d *TWAINDriver) GetScanner(ctx context.Context, scannerID string) (*models.Scanner, error) {
	scanner, ok := d.scanners[scannerID]
	if !ok {
		return nil, fmt.Errorf("scanner not found: %s", scannerID)
	}
	return scanner, nil
}

func (d *TWAINDriver) Scan(ctx context.Context, scannerID string, params models.ScanParams, progressCallback func(int)) ([]models.ScanResult, error) {
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

	// Create output directory
	outputDir := "./scans"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	if progressCallback != nil {
		progressCallback(50)
	}

	// For now, return a placeholder result
	// Full TWAIN implementation would require extensive COM interface work
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("scan_%s.jpg", timestamp)
	filepath := filepath.Join(outputDir, filename)

	result := models.ScanResult{
		PageNumber: 1,
		FilePath:   filepath,
		FileSize:   0,
		Format:     "JPEG",
		Width:      params.Width,
		Height:     params.Height,
	}

	if progressCallback != nil {
		progressCallback(100)
	}

	return []models.ScanResult{result}, nil
}

func (d *TWAINDriver) CancelScan(ctx context.Context, scannerID string) error {
	scanner, err := d.GetScanner(ctx, scannerID)
	if err != nil {
		return err
	}
	scanner.Status = "idle"
	return nil
}

func (d *TWAINDriver) WatchLidStatus(ctx context.Context, scannerID string, callback func(lidClosed bool)) error {
	return fmt.Errorf("lid status monitoring not supported on TWAIN")
}

func (d *TWAINDriver) Close() error {
	if d.initialized && d.dsmLib != nil {
		d.dsmLib.Release()
		d.initialized = false
	}
	return nil
}
