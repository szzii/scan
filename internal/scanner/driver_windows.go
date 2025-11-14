//go:build windows

package scanner

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/nfnt/resize"
	"github.com/scanserver/scanner-service/pkg/models"
)

// WIA Constants (based on NAPS2 implementation)
const (
	WiaDeviceTypeScanner = 1
	WiaIntentColorScan   = 1
	WiaIntentGrayScan    = 2
	WiaIntentTextScan    = 4
	WiaFormatBMP         = "{B96B3CAB-0728-11D3-9D7B-0000F81EF32E}"
	WiaFormatJPEG        = "{B96B3CAE-0728-11D3-9D7B-0000F81EF32E}"
	WiaFormatPNG         = "{B96B3CAF-0728-11D3-9D7B-0000F81EF32E}"
	WiaFormatTIFF        = "{B96B3CB1-0728-11D3-9D7B-0000F81EF32E}"
)

// WIA Property IDs - Complete set from NAPS2
const (
	// Device Properties (WIA 1.0 - DPS = Device Property Set)
	WIA_DPS_DOCUMENT_HANDLING_CAPABILITIES = 3086 // 0x0C0E - Device capabilities (read-only)
	WIA_DPS_DOCUMENT_HANDLING_STATUS       = 3087 // 0x0C0F - Current status (read-only)
	WIA_DPS_DOCUMENT_HANDLING_SELECT       = 3088 // 0x0C10 - Select mode (FEED/FLATBED/DUPLEX)
	WIA_DPS_PAGES                          = 3096 // 0x0C18 - Number of pages to scan
	WIA_DPS_HORIZONTAL_BED_SIZE            = 3074 // 0x0C02 - Flatbed width (1/1000 inch)
	WIA_DPS_VERTICAL_BED_SIZE              = 3075 // 0x0C03 - Flatbed height (1/1000 inch)
	WIA_DPS_HORIZONTAL_SHEET_FEED_SIZE     = 3076 // 0x0C04 - Feeder width (1/1000 inch)
	WIA_DPS_VERTICAL_SHEET_FEED_SIZE       = 3077 // 0x0C05 - Feeder height (1/1000 inch)

	// Item Properties (WIA 2.0 - IPS = Item Property Set)
	WIA_IPS_PAGES                    = 3096 // 0x0C18 - Number of pages (same as DPS)
	WIA_IPS_DOCUMENT_HANDLING_SELECT = 3088 // 0x0C10 - Document handling (WIA 2.0)
	WIA_IPS_MAX_HORIZONTAL_SIZE      = 6165 // 0x1815 - Max width (WIA 2.0)
	WIA_IPS_MAX_VERTICAL_SIZE        = 6166 // 0x1816 - Max height (WIA 2.0)

	// Common Item Properties (IPA = Item Property All)
	WIA_IPA_DATATYPE    = 4103 // 0x1007 - Data type (0=B&W, 2=Grayscale, 3=Color)
	WIA_IPA_BUFFER_SIZE = 4104 // 0x1008 - Transfer buffer size
	WIA_IPA_FORMAT      = 4106 // 0x100A - Image format GUID
	WIA_IPA_TYMED       = 4108 // 0x100C - Transfer method

	// Scanner Item Properties (IPS = Item Property Scanner)
	WIA_IPS_XRES        = 6147 // 0x1803 - Horizontal resolution (DPI)
	WIA_IPS_YRES        = 6148 // 0x1804 - Vertical resolution (DPI)
	WIA_IPS_XPOS        = 6149 // 0x1805 - Horizontal start position
	WIA_IPS_YPOS        = 6150 // 0x1806 - Vertical start position
	WIA_IPS_XEXTENT     = 6151 // 0x1807 - Horizontal extent (width)
	WIA_IPS_YEXTENT     = 6152 // 0x1808 - Vertical extent (height)
	WIA_IPS_BRIGHTNESS  = 6154 // 0x180A - Brightness (-1000 to 1000)
	WIA_IPS_CONTRAST    = 6155 // 0x180B - Contrast (-1000 to 1000)
	WIA_IPS_PREVIEW     = 3100 // 0x0C1C - Preview mode (0=final, 1=preview)
	WIA_IPS_AUTO_DESKEW = 3107 // 0x0C23 - Auto deskew
	WIA_IPS_BLANK_PAGES = 4167 // 0x1047 - Blank page detection
	WIA_IPS_CUR_INTENT  = 6146 // 0x1802 - Current intent (simplified mode)
)

// WIA Property Values - Document Handling
const (
	WIA_USE_FEEDER  = 0x001 // FEED - Use feeder
	WIA_USE_FLATBED = 0x002 // FLAT - Use flatbed
	WIA_USE_DUPLEX  = 0x004 // DUPLEX - Duplex scanning
	WIA_DETECT_FEED = 0x008 // DETECT - Paper detection
)

// WIA Data Types
const (
	WIA_DATA_THRESHOLD = 0 // Black & White (1-bit)
	WIA_DATA_GRAYSCALE = 2 // Grayscale (8-bit)
	WIA_DATA_COLOR     = 3 // Color (24-bit RGB)
)

// WIA Error Codes (HRESULT)
const (
	WIA_ERROR_PAPER_EMPTY   = 0x80210003 // Paper empty
	WIA_ERROR_PAPER_JAM     = 0x80210002 // Paper jam
	WIA_ERROR_OFFLINE       = 0x80210005 // Device offline
	WIA_ERROR_BUSY          = 0x80210006 // Device busy
	WIA_ERROR_WARMING_UP    = 0x80210007 // Device warming up
	WIA_ERROR_COVER_OPEN    = 0x80210016 // Cover open
	WIA_ERROR_DEVICE_LOCKED = 0x8021000A // Device locked
	WIA_ERROR_NO_DEVICE     = 0x80210015 // No device available
	WIA_ERROR_GENERAL_ERROR = 0x80210001 // General error
	WIA_ERROR_NO_MORE_ITEMS = 0x00210001 // Undocumented: no more pages
)

// WindowsDriver implements ScannerDriver for Windows using WIA (Windows Image Acquisition)
type WindowsDriver struct {
	scanners    map[string]*models.Scanner
	deviceMgr   *ole.IDispatch
	initialized bool
}

// newPlatformDriver creates a combined WIA/TWAIN driver for Windows
func newPlatformDriver() (ScannerDriver, error) {
	return newCombinedDriver()
}

// newWIADriverInternal creates a WIA-only driver
func newWIADriverInternal() (*WindowsDriver, error) {
	driver := &WindowsDriver{
		scanners: make(map[string]*models.Scanner),
	}

	// Initialize COM
	err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize COM: %w", err)
	}

	// Try both WIA versions to detect all devices
	var deviceMgr *ole.IDispatch
	var wiaVersion string

	// Try WIA 1.0 first (most compatible, includes D2800+)
	unknown, err := oleutil.CreateObject("WIA.DeviceManager")
	if err == nil {
		deviceMgr, err = unknown.QueryInterface(ole.IID_IDispatch)
		if err == nil {
			wiaVersion = "1.0"
			fmt.Println("WIA: Using WIA 1.0 DeviceManager")
		}
	}

	// If WIA 1.0 failed, try WIA 2.0
	if deviceMgr == nil {
		fmt.Println("WIA: WIA 1.0 failed, trying WIA 2.0...")
		unknown, err = oleutil.CreateObject("WIA.DeviceManager.1")
		if err == nil {
			deviceMgr, err = unknown.QueryInterface(ole.IID_IDispatch)
			if err == nil {
				wiaVersion = "2.0"
				fmt.Println("WIA: Using WIA 2.0 DeviceManager")
			}
		}
	}

	if deviceMgr == nil {
		ole.CoUninitialize()
		return nil, fmt.Errorf("failed to create WIA DeviceManager (tried both 1.0 and 2.0): %w", err)
	}

	fmt.Printf("WIA: DeviceManager %s initialized successfully\n", wiaVersion)
	driver.deviceMgr = deviceMgr
	driver.initialized = true

	return driver, nil
}

func (d *WindowsDriver) ListScanners(ctx context.Context) ([]models.Scanner, error) {
	if !d.initialized {
		return nil, fmt.Errorf("driver not initialized")
	}

	fmt.Println("WIA: Starting scanner enumeration...")
	fmt.Println("WIA: Attempting to enumerate all WIA devices...")

	// Method 1: Try with device type filter (recommended)
	deviceInfosRaw, err := oleutil.CallMethod(d.deviceMgr, "DeviceInfos", WiaDeviceTypeScanner)
	if err != nil {
		// Method 2: Try without filter (gets all device types)
		fmt.Printf("WIA: Filtered DeviceInfos failed: %v\n", err)
		fmt.Println("WIA: Trying unfiltered DeviceInfos (all device types)...")
		deviceInfosRaw, err = oleutil.GetProperty(d.deviceMgr, "DeviceInfos")
		if err != nil {
			return nil, fmt.Errorf("failed to get DeviceInfos: %w", err)
		}
	} else {
		fmt.Println("WIA: Using filtered DeviceInfos (scanners only)")
	}

	deviceInfos := deviceInfosRaw.ToIDispatch()
	defer deviceInfos.Release()

	// Get device count
	countRaw, err := oleutil.GetProperty(deviceInfos, "Count")
	if err != nil {
		return nil, fmt.Errorf("failed to get device count: %w", err)
	}
	count := int(countRaw.Val)

	fmt.Printf("WIA: Found %d device(s) in DeviceInfos collection\n", count)

	var scanners []models.Scanner
	var allDevices []string

	// Enumerate devices
	for i := 1; i <= count; i++ {
		fmt.Printf("WIA: Checking device %d/%d...\n", i, count)

		deviceInfoRaw, err := oleutil.GetProperty(deviceInfos, "Item", i)
		if err != nil {
			fmt.Printf("WIA: Failed to get device %d: %v\n", i, err)
			continue
		}
		deviceInfo := deviceInfoRaw.ToIDispatch()

		// Get device type
		deviceTypeRaw, err := oleutil.GetProperty(deviceInfo, "Type")
		if err != nil {
			fmt.Printf("WIA: Failed to get device type for device %d: %v\n", i, err)
			deviceInfo.Release()
			continue
		}
		deviceType := int(deviceTypeRaw.Val)

		// Get device ID for logging
		deviceIDRaw, err := oleutil.GetProperty(deviceInfo, "DeviceID")
		var deviceID string
		if err == nil {
			deviceID = deviceIDRaw.ToString()
		} else {
			deviceID = fmt.Sprintf("unknown-%d", i)
		}

		// Get device name for logging
		deviceName := "Unknown"
		propsRaw, err := oleutil.GetProperty(deviceInfo, "Properties")
		if err == nil {
			props := propsRaw.ToIDispatch()
			namePropRaw, err := oleutil.GetProperty(props, "Item", "Name")
			if err == nil {
				nameProp := namePropRaw.ToIDispatch()
				nameValueRaw, err := oleutil.GetProperty(nameProp, "Value")
				if err == nil {
					deviceName = nameValueRaw.ToString()
				}
				nameProp.Release()
			}
			props.Release()
		}

		// Log all devices found
		allDevices = append(allDevices, fmt.Sprintf("%s (Type: %d, ID: %s)", deviceName, deviceType, deviceID))
		fmt.Printf("WIA: Device %d: Name='%s', Type=%d, ID='%s'\n", i, deviceName, deviceType, deviceID)

		// Only include scanners (Type = 1)
		if deviceType != WiaDeviceTypeScanner {
			fmt.Printf("WIA: Skipping device (not a scanner, type=%d)\n", deviceType)
			deviceInfo.Release()
			continue
		}

		// Get device name
		name := deviceName
		if name == "Unknown" {
			name = "Unknown Scanner"
		}

		// Get manufacturer
		manufacturer := "Unknown"
		propsRaw2, err := oleutil.GetProperty(deviceInfo, "Properties")
		if err == nil {
			props := propsRaw2.ToIDispatch()
			mfgPropRaw, err := oleutil.GetProperty(props, "Item", "Manufacturer")
			if err == nil {
				mfgProp := mfgPropRaw.ToIDispatch()
				mfgValueRaw, err := oleutil.GetProperty(mfgProp, "Value")
				if err == nil {
					manufacturer = mfgValueRaw.ToString()
				}
				mfgProp.Release()
			}
			props.Release()
		}

		scanner := models.Scanner{
			ID:           deviceID,
			Name:         name,
			Model:        name,
			Manufacturer: manufacturer,
			Status:       "idle",
			Capabilities: models.Capability{
				MaxWidth:        2100, // A4 width in 0.1mm (210mm)
				MaxHeight:       2970, // A4 height in 0.1mm (297mm)
				Resolutions:     []int{75, 100, 150, 200, 300, 600},
				ColorModes:      []string{"Color", "Grayscale", "BlackAndWhite"},
				DocumentFormats: []string{"JPEG", "PNG", "TIFF", "BMP"},
				FeederEnabled:   false,
				DuplexEnabled:   false,
			},
			LastSeen: time.Now(),
		}

		d.scanners[scanner.ID] = &scanner
		scanners = append(scanners, scanner)

		fmt.Printf("WIA: ✓ Added scanner: %s (Manufacturer: %s)\n", name, manufacturer)

		deviceInfo.Release()
	}

	fmt.Printf("WIA: Enumeration complete. Found %d scanner(s) out of %d device(s)\n", len(scanners), count)
	if len(allDevices) > 0 {
		fmt.Println("WIA: All devices found:")
		for idx, dev := range allDevices {
			fmt.Printf("  %d. %s\n", idx+1, dev)
		}
	}

	if len(scanners) == 0 {
		return nil, fmt.Errorf("no WIA scanners found (found %d non-scanner device(s))", len(allDevices))
	}

	return scanners, nil
}

func (d *WindowsDriver) GetScanner(ctx context.Context, scannerID string) (*models.Scanner, error) {
	scanner, ok := d.scanners[scannerID]
	if !ok {
		return nil, fmt.Errorf("scanner not found: %s", scannerID)
	}
	return scanner, nil
}

func (d *WindowsDriver) Scan(ctx context.Context, scannerID string, params models.ScanParams, progressCallback func(int)) ([]models.ScanResult, error) {
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

	// Get DeviceInfos collection
	deviceInfosRaw, err := oleutil.GetProperty(d.deviceMgr, "DeviceInfos")
	if err != nil {
		return nil, fmt.Errorf("failed to get DeviceInfos: %w", err)
	}
	deviceInfos := deviceInfosRaw.ToIDispatch()
	defer deviceInfos.Release()

	// Find the device by ID
	countRaw, err := oleutil.GetProperty(deviceInfos, "Count")
	if err != nil {
		return nil, fmt.Errorf("failed to get device count: %w", err)
	}
	count := int(countRaw.Val)

	var deviceInfo *ole.IDispatch
	for i := 1; i <= count; i++ {
		devInfoRaw, err := oleutil.GetProperty(deviceInfos, "Item", i)
		if err != nil {
			continue
		}
		devInfo := devInfoRaw.ToIDispatch()

		deviceIDRaw, err := oleutil.GetProperty(devInfo, "DeviceID")
		if err != nil {
			devInfo.Release()
			continue
		}
		deviceID := deviceIDRaw.ToString()

		if deviceID == scannerID {
			deviceInfo = devInfo
			break
		}
		devInfo.Release()
	}

	if deviceInfo == nil {
		return nil, fmt.Errorf("scanner not found: %s", scannerID)
	}
	defer deviceInfo.Release()

	// Connect to device
	deviceRaw, err := oleutil.CallMethod(deviceInfo, "Connect")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to scanner: %w", err)
	}
	device := deviceRaw.ToIDispatch()
	defer device.Release()

	// Get scanner item
	itemsRaw, err := oleutil.GetProperty(device, "Items")
	if err != nil {
		return nil, fmt.Errorf("failed to get device items: %w", err)
	}
	items := itemsRaw.ToIDispatch()
	defer items.Release()

	itemRaw, err := oleutil.GetProperty(items, "Item", 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get scanner item: %w", err)
	}
	item := itemRaw.ToIDispatch()
	defer item.Release()

	// Set scan properties
	propsRaw, err := oleutil.GetProperty(item, "Properties")
	if err != nil {
		return nil, fmt.Errorf("failed to get properties: %w", err)
	}
	props := propsRaw.ToIDispatch()
	defer props.Release()

	// Configure scan properties using NAPS2's SafeSetProperty pattern
	// This ensures we don't fail if a scanner doesn't support certain properties

	fmt.Println("Configuring WIA properties (NAPS2 mode)...")

	// Set data type (color mode) - NAPS2 line 426-438
	var dataType int
	switch params.ColorMode {
	case "BlackAndWhite":
		dataType = WIA_DATA_THRESHOLD // 0 - B&W
	case "Grayscale":
		dataType = WIA_DATA_GRAYSCALE // 2 - Grayscale
	case "Color":
		dataType = WIA_DATA_COLOR // 3 - Color
	default:
		dataType = WIA_DATA_COLOR
	}
	d.safeSetPropertyInt(props, WIA_IPA_DATATYPE, dataType)
	fmt.Printf("  Data type: %d (%s)\n", dataType, params.ColorMode)

	// Set resolution (DPI) - NAPS2 line 463-465
	d.safeSetPropertyInt(props, WIA_IPS_XRES, params.Resolution)
	d.safeSetPropertyInt(props, WIA_IPS_YRES, params.Resolution)
	fmt.Printf("  Resolution: %d DPI\n", params.Resolution)

	// Set paper size and alignment (NAPS2 feature) - NAPS2 line 447-474
	if params.PageSize != "" || params.PageWidth > 0 || params.Width > 0 {
		pageWidth, pageHeight, xPos, err := d.calculateScanArea(device, item, params, params.UseFeeder)
		if err == nil {
			// Apply WIA offset width mode if requested (NAPS2 compatibility)
			if params.WiaOffsetWidth {
				// NAPS2 mode: add offset to width
				d.safeSetPropertyInt(props, WIA_IPS_XEXTENT, pageWidth+xPos)
				d.safeSetPropertyInt(props, WIA_IPS_XPOS, xPos)
			} else {
				// Standard mode: separate width and position
				d.safeSetPropertyInt(props, WIA_IPS_XEXTENT, pageWidth)
				d.safeSetPropertyInt(props, WIA_IPS_XPOS, xPos)
			}
			d.safeSetPropertyInt(props, WIA_IPS_YEXTENT, pageHeight)
			d.safeSetPropertyInt(props, WIA_IPS_YPOS, 0) // Always start from top

			fmt.Printf("  Scan area: %dx%d pixels at offset (%d, 0)\n", pageWidth, pageHeight, xPos)
		} else {
			fmt.Printf("  Warning: Failed to calculate scan area: %v\n", err)
		}
	}

	// Set scan intent (simplified mode setting)
	intent := WiaIntentColorScan
	if params.ColorMode == "Grayscale" {
		intent = WiaIntentGrayScan
	} else if params.ColorMode == "BlackAndWhite" {
		intent = WiaIntentTextScan
	}
	d.safeSetPropertyInt(props, WIA_IPS_CUR_INTENT, intent)

	// Set document handling if using feeder (NAPS2 line 387-420)
	if params.UseFeeder {
		fmt.Println("  ADF mode enabled")

		// Document handling select - FEEDER + DUPLEX + DETECT
		handlingValue := WIA_USE_FEEDER | WIA_DETECT_FEED
		if params.UseDuplex {
			handlingValue |= WIA_USE_DUPLEX
			fmt.Println("  Duplex mode enabled")
		}
		d.safeSetPropertyInt(props, WIA_DPS_DOCUMENT_HANDLING_SELECT, handlingValue)
		fmt.Printf("  Document handling: 0x%03X\n", handlingValue)

		// Set pages to scan - NAPS2 line 377-386
		// WIA 1.0 uses 1 page at a time with looping (we handle this in scanADFBatch)
		// But we still set this to hint to the driver
		if params.PageCount == 0 {
			d.safeSetPropertyInt(props, WIA_DPS_PAGES, 1) // WIA 1.0: scan 1 page per Transfer
			d.safeSetPropertyInt(props, WIA_IPS_PAGES, 0) // WIA 2.0: scan all pages
			fmt.Println("  Pages: ALL (continuous until empty)")
		} else {
			d.safeSetPropertyInt(props, WIA_DPS_PAGES, 1) // Still use 1 for WIA 1.0 loop
			d.safeSetPropertyInt(props, WIA_IPS_PAGES, params.PageCount) // WIA 2.0
			fmt.Printf("  Pages: %d\n", params.PageCount)
		}

		// Preview mode - 0 for final scan (NAPS2 line 440-443)
		d.safeSetPropertyInt(props, WIA_IPS_PREVIEW, 0)

		// Transfer buffer size for performance (NAPS2 optimization)
		d.safeSetPropertyInt(props, WIA_IPA_BUFFER_SIZE, 65536) // 64KB buffer
		fmt.Println("  Buffer size: 64KB")

		// Auto deskew - straighten tilted pages (NAPS2 feature)
		d.safeSetPropertyInt(props, WIA_IPS_AUTO_DESKEW, 1)
		fmt.Println("  Auto deskew: enabled")

		// Blank page detection - skip empty pages (NAPS2 feature)
		d.safeSetPropertyInt(props, WIA_IPS_BLANK_PAGES, 1)
		fmt.Println("  Blank page detection: enabled")
	} else {
		// Flatbed mode
		fmt.Println("  Flatbed mode")
		d.safeSetPropertyInt(props, WIA_DPS_DOCUMENT_HANDLING_SELECT, WIA_USE_FLATBED)
	}

	fmt.Println("Property configuration complete")

	// Create output directory
	outputDir := "./scans"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	var results []models.ScanResult
	pageCount := params.PageCount
	if pageCount == 0 {
		pageCount = 1
	}

	// Generate base timestamp for all pages in this batch
	baseTimestamp := time.Now().Format("20060102_150405")

	// For ADF mode, use optimized batch scanning
	if params.UseFeeder {
		// Try to get all images in one go using WIA's multi-page transfer
		// This tells WIA to buffer all pages during scanning for faster operation
		return d.scanADFBatch(ctx, item, outputDir, baseTimestamp, pageCount, params, progressCallback)
	}

	// Single page or flatbed mode - standard transfer
	imageRaw, err := oleutil.CallMethod(item, "Transfer", WiaFormatJPEG)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer image: %w", err)
	}
	image := imageRaw.ToIDispatch()
	defer image.Release()

	filename := fmt.Sprintf("scan_%s.jpg", baseTimestamp)
	filePath := filepath.Join(outputDir, filename)

	_, err = oleutil.CallMethod(image, "SaveFile", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to save image: %w", err)
	}

	// Post-processing: Apply JPEG quality control (same as ADF batch mode)
	if params.MaxQuality || params.JpegQuality > 0 {
		if err := d.applyImageQuality(filePath, params); err != nil {
			fmt.Printf("  Warning: Image quality adjustment failed: %v\n", err)
		}
	}

	// Post-processing: Scale ratio (NAPS2 feature)
	if params.ScaleRatio > 1 {
		if err := d.applyScaleRatio(filePath, params.ScaleRatio, params); err != nil {
			fmt.Printf("  Warning: Scale ratio failed: %v\n", err)
		}
	}

	// Post-processing: Crop/stretch to page size (NAPS2 feature)
	if params.CropToPageSize || params.StretchToPageSize {
		if err := d.applyCropToPageSize(filePath, params); err != nil {
			fmt.Printf("  Warning: Crop to page size failed: %v\n", err)
		}
	}

	fileInfo, err := os.Stat(filePath)
	fileSize := int64(0)
	if err == nil {
		fileSize = fileInfo.Size()
	}

	result := models.ScanResult{
		PageNumber: 1,
		FilePath:   filePath,
		FileSize:   fileSize,
		Format:     "JPEG",
		Width:      params.Width,
		Height:     params.Height,
	}

	results = append(results, result)

	if progressCallback != nil {
		progressCallback(100)
	}

	return results, nil
}

// scanADFBatch performs optimized batch scanning for ADF mode
// Based on NAPS2's WIA 1.0 implementation: continuously call Transfer until PAPER_EMPTY
func (d *WindowsDriver) scanADFBatch(ctx context.Context, item *ole.IDispatch, outputDir, baseTimestamp string, pageCount int, params models.ScanParams, progressCallback func(int)) ([]models.ScanResult, error) {
	var results []models.ScanResult

	// Channel for async file operations
	type saveTask struct {
		image      *ole.IDispatch
		pageNum    int
		filePath   string
		resultChan chan models.ScanResult
		errChan    chan error
	}

	maxPages := pageCount
	if maxPages == 0 {
		maxPages = 9999 // Scan until feeder is empty
	}

	saveChan := make(chan saveTask, maxPages)
	doneChan := make(chan struct{})

	// Worker goroutine for async file saving and post-processing (NAPS2 pattern)
	// This allows the scanner to continue scanning while we save previous pages
	go func() {
		for task := range saveChan {
			// Save the image file
			_, err := oleutil.CallMethod(task.image, "SaveFile", task.filePath)
			task.image.Release()

			if err != nil {
				task.errChan <- fmt.Errorf("failed to save page %d: %w", task.pageNum, err)
				continue
			}

			// Post-processing: Blank page detection (NAPS2 feature)
			if params.ExcludeBlankPages {
				// Use default thresholds if not specified
				whiteThreshold := params.BlankPageWhiteThreshold
				if whiteThreshold == 0 {
					whiteThreshold = models.DefaultBlankPageWhiteThreshold // 70
				}
				coverageThreshold := params.BlankPageCoverageThreshold
				if coverageThreshold == 0 {
					coverageThreshold = models.DefaultBlankPageCoverageThreshold // 15
				}

				detector := &BlankPageDetector{
					WhiteThreshold:    whiteThreshold,
					CoverageThreshold: coverageThreshold,
				}

				isBlank, err := detector.isBlankPage(task.filePath)
				if err != nil {
					fmt.Printf("  Warning: Blank page detection failed for page %d: %v\n", task.pageNum, err)
				} else if isBlank {
					// Delete blank page
					os.Remove(task.filePath)
					fmt.Printf("  Excluded blank page %d\n", task.pageNum)
					continue // Skip adding to results
				}
			}

			// Post-processing: Scale ratio (NAPS2 feature)
			if params.ScaleRatio > 1 {
				if err := d.applyScaleRatio(task.filePath, params.ScaleRatio, params); err != nil {
					fmt.Printf("  Warning: Scale ratio failed for page %d: %v\n", task.pageNum, err)
				}
			}

			// Post-processing: Crop/stretch to page size (NAPS2 feature)
			if params.CropToPageSize || params.StretchToPageSize {
				if err := d.applyCropToPageSize(task.filePath, params); err != nil {
					fmt.Printf("  Warning: Crop to page size failed for page %d: %v\n", task.pageNum, err)
				}
			}

			// Post-processing: Image quality control (NAPS2 feature)
			if params.MaxQuality || params.JpegQuality > 0 {
				if err := d.applyImageQuality(task.filePath, params); err != nil {
					fmt.Printf("  Warning: Image quality adjustment failed for page %d: %v\n", task.pageNum, err)
				}
			}

			// Get file info
			fileInfo, err := os.Stat(task.filePath)
			fileSize := int64(0)
			if err == nil {
				fileSize = fileInfo.Size()
			}

			result := models.ScanResult{
				PageNumber: task.pageNum,
				FilePath:   task.filePath,
				FileSize:   fileSize,
				Format:     "JPEG",
				Width:      params.Width,
				Height:     params.Height,
			}

			task.resultChan <- result
		}
		close(doneChan)
	}()

	resultChan := make(chan models.ScanResult, maxPages)
	errChan := make(chan error, maxPages)

	// NAPS2's core technique: Loop Transfer calls until PAPER_EMPTY
	// This is the key to WIA 1.0 batch scanning performance
	scannedPages := 0
	fmt.Println("Starting WIA batch scanning loop (NAPS2 mode)...")

	for i := 0; i < maxPages; i++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			close(saveChan)
			<-doneChan
			return results, ctx.Err()
		default:
		}

		// Update progress
		if progressCallback != nil && pageCount > 0 {
			progress := (i * 50) / pageCount // 0-50% for scanning
			progressCallback(progress)
		}

		// Transfer image - this is the hardware scan operation
		// WIA will block here until the page is scanned
		fmt.Printf("Calling Transfer for page %d...\n", i+1)
		imageRaw, err := oleutil.CallMethod(item, "Transfer", WiaFormatJPEG)

		if err != nil {
			// Check if it's PAPER_EMPTY error (expected when done)
			if isWiaError(err, WIA_ERROR_PAPER_EMPTY) {
				fmt.Printf("Feeder empty after %d pages (normal)\n", scannedPages)
				break
			}

			// Check for NO_MORE_ITEMS (undocumented, seen in NAPS2)
			if isWiaError(err, WIA_ERROR_NO_MORE_ITEMS) {
				fmt.Printf("No more items after %d pages (normal)\n", scannedPages)
				break
			}

			// First page failure is an error
			if i == 0 {
				close(saveChan)
				<-doneChan
				return nil, handleWiaError(err)
			}

			// Subsequent page errors might just mean we're done
			fmt.Printf("Transfer error after %d pages: %v\n", scannedPages, err)
			break
		}

		image := imageRaw.ToIDispatch()

		// Check for empty stream (NAPS2 pattern - line 254-257)
		// Some scanners return success but empty image
		if image == nil {
			fmt.Println("Warning: Received nil image from Transfer")
			break
		}

		scannedPages++
		fmt.Printf("Successfully scanned page %d\n", scannedPages)

		// Generate filename
		filename := fmt.Sprintf("scan_%s_page_%d.jpg", baseTimestamp, scannedPages)
		filePath := filepath.Join(outputDir, filename)

		// Send to async saver immediately (NAPS2 async pattern)
		// This allows scanner to start next page while we save current one
		saveChan <- saveTask{
			image:      image,
			pageNum:    scannedPages,
			filePath:   filePath,
			resultChan: resultChan,
			errChan:    errChan,
		}

		// Continue looping - next Transfer call will scan next page
		// This is the NAPS2 WIA 1.0 batch scanning secret!
	}

	fmt.Printf("Scanning phase complete. Scanned %d pages total.\n", scannedPages)

	// Close save channel and wait for all saves to complete
	close(saveChan)
	<-doneChan

	// Collect all results
	for i := 0; i < scannedPages; i++ {
		select {
		case result := <-resultChan:
			results = append(results, result)
			// Update progress for saving phase
			if progressCallback != nil {
				progress := 50 + ((i+1)*50)/scannedPages // 50-100% for saving
				progressCallback(progress)
			}
		case err := <-errChan:
			return results, err
		case <-ctx.Done():
			return results, ctx.Err()
		}
	}

	if progressCallback != nil {
		progressCallback(100)
	}

	fmt.Printf("Batch scanning complete: %d pages saved successfully\n", len(results))
	return results, nil
}

func (d *WindowsDriver) setProperty(props *ole.IDispatch, propID string, value interface{}) error {
	propRaw, err := oleutil.GetProperty(props, "Item", propID)
	if err != nil {
		return err
	}
	prop := propRaw.ToIDispatch()
	defer prop.Release()

	_, err = oleutil.PutProperty(prop, "Value", value)
	return err
}

// safeSetPropertyInt sets a property by integer ID, logging errors but not failing
// This matches NAPS2's SafeSetProperty pattern
func (d *WindowsDriver) safeSetPropertyInt(props *ole.IDispatch, propID int, value interface{}) {
	propIDStr := fmt.Sprintf("%d", propID)
	err := d.setProperty(props, propIDStr, value)
	if err != nil {
		// Log but don't fail - property might not be supported
		fmt.Printf("Warning: Could not set property %d (0x%X): %v\n", propID, propID, err)
	}
}

// safeSetProperty sets a property by string ID, logging errors but not failing
func (d *WindowsDriver) safeSetProperty(props *ole.IDispatch, propID string, value interface{}) {
	err := d.setProperty(props, propID, value)
	if err != nil {
		fmt.Printf("Warning: Could not set property %s: %v\n", propID, err)
	}
}

// isWiaError checks if an error is a specific WIA error code
func isWiaError(err error, errorCode uint32) bool {
	if err == nil {
		return false
	}
	// Check if error message contains the hex code
	errMsg := err.Error()
	hexCode := fmt.Sprintf("0x%X", errorCode)
	return len(errMsg) > 0 && (errMsg == hexCode || len(errMsg) > len(hexCode) && errMsg[len(errMsg)-len(hexCode):] == hexCode)
}

// handleWiaError converts WIA error codes to user-friendly messages
func handleWiaError(err error) error {
	if err == nil {
		return nil
	}

	// Check for specific WIA error codes
	if isWiaError(err, WIA_ERROR_PAPER_EMPTY) {
		return fmt.Errorf("feeder is empty - no more pages to scan")
	}
	if isWiaError(err, WIA_ERROR_PAPER_JAM) {
		return fmt.Errorf("paper jam detected")
	}
	if isWiaError(err, WIA_ERROR_OFFLINE) {
		return fmt.Errorf("scanner is offline")
	}
	if isWiaError(err, WIA_ERROR_BUSY) {
		return fmt.Errorf("scanner is busy")
	}
	if isWiaError(err, WIA_ERROR_WARMING_UP) {
		return fmt.Errorf("scanner is warming up")
	}
	if isWiaError(err, WIA_ERROR_COVER_OPEN) {
		return fmt.Errorf("scanner cover is open")
	}
	if isWiaError(err, WIA_ERROR_NO_MORE_ITEMS) {
		return fmt.Errorf("no more pages available")
	}

	return err
}

func (d *WindowsDriver) CancelScan(ctx context.Context, scannerID string) error {
	scanner, err := d.GetScanner(ctx, scannerID)
	if err != nil {
		return err
	}

	scanner.Status = "idle"
	return nil
}

func (d *WindowsDriver) WatchLidStatus(ctx context.Context, scannerID string, callback func(lidClosed bool)) error {
	// WIA doesn't support lid status monitoring
	// This is a stub for interface compatibility
	return fmt.Errorf("lid status monitoring not supported on WIA")
}

func (d *WindowsDriver) Close() error {
	if d.initialized {
		if d.deviceMgr != nil {
			d.deviceMgr.Release()
		}
		ole.CoUninitialize()
		d.initialized = false
	}
	return nil
}

// calculateScanArea calculates scan area in pixels based on page size settings
// Implements NAPS2's paper size calculation algorithm (WiaScanDriver.cs:447-474)
func (d *WindowsDriver) calculateScanArea(
	device *ole.IDispatch,
	item *ole.IDispatch,
	params models.ScanParams,
	useFeeder bool,
) (width, height, xPos int, err error) {
	// 1. Get page dimensions in millimeters
	var pageWidthMM, pageHeightMM int

	if params.PageSize != "" && params.PageSize != "Custom" {
		// Use predefined paper size
		if size, ok := models.PaperSizes[params.PageSize]; ok {
			pageWidthMM = size.Width
			pageHeightMM = size.Height
		} else {
			return 0, 0, 0, fmt.Errorf("unknown page size: %s", params.PageSize)
		}
	} else {
		// Use custom dimensions
		pageWidthMM = params.PageWidth
		pageHeightMM = params.PageHeight

		// Fallback to legacy Width/Height fields for backward compatibility
		if pageWidthMM == 0 {
			pageWidthMM = params.Width
		}
		if pageHeightMM == 0 {
			pageHeightMM = params.Height
		}

		// Default to A4 if no dimensions specified
		if pageWidthMM == 0 && pageHeightMM == 0 {
			pageWidthMM = 210 // A4 width
			pageHeightMM = 297 // A4 height
		}
	}

	// 2. Convert millimeters to pixels using NAPS2's formula
	// Formula: pixels = (mm / 25.4) * DPI
	// 25.4 mm = 1 inch
	resolution := params.Resolution
	if resolution == 0 {
		resolution = 300 // Default DPI
	}

	pageWidthPixels := int(float64(pageWidthMM) / 25.4 * float64(resolution))
	pageHeightPixels := int(float64(pageHeightMM) / 25.4 * float64(resolution))

	fmt.Printf("  Page size: %dx%d mm = %dx%d pixels @ %d DPI\n",
		pageWidthMM, pageHeightMM, pageWidthPixels, pageHeightPixels, resolution)

	// 3. Calculate horizontal alignment if requested
	xPos = 0
	if params.PageAlign != "" {
		// Get maximum scan width from device
		maxWidthPixels := d.getMaxScanWidth(device, item, useFeeder, resolution)
		if maxWidthPixels > 0 && maxWidthPixels > pageWidthPixels {
			xPos = d.calculateHorizontalAlignment(pageWidthPixels, maxWidthPixels, params.PageAlign)
			fmt.Printf("  Horizontal align: %s (xPos=%d, maxWidth=%d)\n",
				params.PageAlign, xPos, maxWidthPixels)
		}
	}

	return pageWidthPixels, pageHeightPixels, xPos, nil
}

// calculateHorizontalAlignment calculates horizontal start position for alignment
// Implements NAPS2's alignment algorithm (WiaScanDriver.cs:455-459)
func (d *WindowsDriver) calculateHorizontalAlignment(
	pageWidth, maxWidth int,
	alignment string,
) int {
	switch alignment {
	case models.AlignCenter:
		// Center alignment: start at middle
		return (maxWidth - pageWidth) / 2

	case models.AlignLeft:
		// Left alignment: start at left edge
		return maxWidth - pageWidth

	case models.AlignRight:
		fallthrough
	default:
		// Right alignment: start at position 0 (default)
		return 0
	}
}

// getMaxScanWidth retrieves the maximum scan width from device properties
// Reads WIA device properties to determine hardware limits
func (d *WindowsDriver) getMaxScanWidth(
	device *ole.IDispatch,
	item *ole.IDispatch,
	useFeeder bool,
	resolution int,
) int {
	// Try to get device properties
	propsRaw, err := oleutil.GetProperty(device, "Properties")
	if err != nil {
		return 0
	}
	props := propsRaw.ToIDispatch()
	defer props.Release()

	// Try WIA 2.0 property first (IPS_MAX_HORIZONTAL_SIZE)
	if propRaw, err := oleutil.GetProperty(props, "Item", fmt.Sprintf("%d", WIA_IPS_MAX_HORIZONTAL_SIZE)); err == nil {
		prop := propRaw.ToIDispatch()
		if valueRaw, err := oleutil.GetProperty(prop, "Value"); err == nil {
			maxWidth := int(valueRaw.Val)
			prop.Release()
			fmt.Printf("  Max scan width (WIA 2.0): %d pixels\n", maxWidth)
			return maxWidth
		}
		prop.Release()
	}

	// Try WIA 1.0 properties (DPS_HORIZONTAL_BED_SIZE or DPS_HORIZONTAL_SHEET_FEED_SIZE)
	// These are in 1/1000 inch units
	var propID int
	if useFeeder {
		propID = WIA_DPS_HORIZONTAL_SHEET_FEED_SIZE // Feeder width
	} else {
		propID = WIA_DPS_HORIZONTAL_BED_SIZE // Flatbed width
	}

	if propRaw, err := oleutil.GetProperty(props, "Item", fmt.Sprintf("%d", propID)); err == nil {
		prop := propRaw.ToIDispatch()
		if valueRaw, err := oleutil.GetProperty(prop, "Value"); err == nil {
			// Convert from 1/1000 inch to pixels
			// Formula: pixels = (thousandths_inch / 1000) * DPI
			maxWidthThousandths := int(valueRaw.Val)
			maxWidthPixels := (maxWidthThousandths * resolution) / 1000
			prop.Release()
			fmt.Printf("  Max scan width (WIA 1.0): %d pixels (%d/1000 inch)\n",
				maxWidthPixels, maxWidthThousandths)
			return maxWidthPixels
		}
		prop.Release()
	}

	// Fallback: assume A4 width (210mm) as maximum
	defaultMaxWidth := int(210.0 / 25.4 * float64(resolution))
	fmt.Printf("  Max scan width (default A4): %d pixels\n", defaultMaxWidth)
	return defaultMaxWidth
}

// BlankPageDetector detects blank pages using NAPS2's YUV luma algorithm
// Implements NAPS2's blank page detection (BlankDetectionImageOp.cs)
type BlankPageDetector struct {
	WhiteThreshold    int // 0-100 (default: 70) - brightness threshold for "white"
	CoverageThreshold int // 0-100 (default: 15) - percentage of non-white pixels
}

// isBlankPage detects if an image is a blank page
// Uses NAPS2's YUV luma algorithm for accurate detection
func (d *BlankPageDetector) isBlankPage(imagePath string) (bool, error) {
	// 1. Open and decode image
	file, err := os.Open(imagePath)
	if err != nil {
		return false, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return false, fmt.Errorf("failed to decode image: %w", err)
	}

	// 2. Calculate adjusted thresholds using NAPS2's formulas
	// whiteThresholdAdjusted = 1 + (whiteThreshold / 100.0) * 254
	// Example: whiteThreshold=70 -> 179
	whiteThresholdAdjusted := 1 + int(float64(d.WhiteThreshold)/100.0*254)

	// coverageThresholdAdjusted = 0.00 + (coverageThreshold / 100.0) * 0.01
	// Example: coverageThreshold=15 -> 0.0015 (0.15%)
	coverageThresholdAdjusted := 0.00 + (float64(d.CoverageThreshold)/100.0)*0.01

	// 3. Ignore 1% edge area to avoid border effects (NAPS2 pattern)
	bounds := img.Bounds()
	ignoreEdge := int(float64(bounds.Dx()) * 0.01)
	if ignoreEdge < 1 {
		ignoreEdge = 0
	}

	startX := bounds.Min.X + ignoreEdge
	endX := bounds.Max.X - ignoreEdge
	startY := bounds.Min.Y + ignoreEdge
	endY := bounds.Max.Y - ignoreEdge

	// Ensure valid bounds
	if startX >= endX || startY >= endY {
		startX = bounds.Min.X
		endX = bounds.Max.X
		startY = bounds.Min.Y
		endY = bounds.Max.Y
	}

	// 4. Scan pixels and calculate coverage
	totalPixels := (endX - startX) * (endY - startY)
	nonWhitePixels := 0

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// Convert from 16-bit to 8-bit
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// YUV luma formula (NAPS2: r*299 + g*587 + b*114)
			// This is the standard ITU-R BT.601 luma calculation
			// Multiplied by 1000 to avoid floating point (NAPS2 pattern)
			luma := int(r8)*299 + int(g8)*587 + int(b8)*114

			// Check if pixel is non-white
			// luma < whiteThresholdAdjusted * 1000
			if luma < whiteThresholdAdjusted*1000 {
				nonWhitePixels++
			}
		}
	}

	// 5. Calculate coverage ratio
	coverage := float64(nonWhitePixels) / float64(totalPixels)

	// 6. Determine if blank
	isBlank := coverage < coverageThresholdAdjusted

	fmt.Printf("  Blank page detection: coverage=%.4f%%, threshold=%.4f%%, blank=%v\n",
		coverage*100, coverageThresholdAdjusted*100, isBlank)

	return isBlank, nil
}

// applyImageQuality applies quality settings to a saved image
// Implements NAPS2's image quality control (MaxQuality and JPEG compression)
func (d *WindowsDriver) applyImageQuality(imagePath string, params models.ScanParams) error {
	// Recompress if quality settings are specified
	// This ensures we apply the user's chosen quality instead of WIA's default
	shouldRecompress := false

	if params.MaxQuality {
		shouldRecompress = true
	} else if params.JpegQuality > 0 {
		// Always recompress if user specified a quality (even if it's the default 75)
		// because WIA's SaveFile may use a different default quality
		shouldRecompress = true
	}

	if !shouldRecompress {
		return nil // No quality adjustment needed
	}

	// 1. Open and decode image
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}
	file.Close() // Close early to allow overwriting

	// 2. Determine output format and quality
	var outputPath string
	var saveErr error

	if params.MaxQuality {
		// Lossless PNG encoding
		outputPath = imagePath
		// Change extension to .png if it's not already
		if filepath.Ext(outputPath) == ".jpg" || filepath.Ext(outputPath) == ".jpeg" {
			outputPath = outputPath[:len(outputPath)-len(filepath.Ext(outputPath))] + ".png"
		}

		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outFile.Close()

		saveErr = savePNG(img, outFile)
		if saveErr == nil && outputPath != imagePath {
			// Remove old JPEG file if we created a new PNG
			os.Remove(imagePath)
			fmt.Printf("  Saved as lossless PNG (MaxQuality): %s\n", outputPath)
		}
	} else {
		// JPEG compression with specified quality
		quality := params.JpegQuality
		if quality == 0 {
			quality = models.DefaultJpegQuality // 75
		}

		outFile, err := os.Create(imagePath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer outFile.Close()

		saveErr = saveJPEG(img, outFile, quality)
		fmt.Printf("  Recompressed JPEG (quality=%d, original=%s)\n", quality, format)
	}

	return saveErr
}

// saveJPEG saves an image as JPEG with specified quality
func saveJPEG(img image.Image, w io.Writer, quality int) error {
	return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
}

// savePNG saves an image as PNG (lossless)
func savePNG(img image.Image, w io.Writer) error {
	return png.Encode(w, img)
}

// applyScaleRatio scales an image by the specified ratio
// Implements NAPS2's scale transformation (1:1, 1:2, 1:4, 1:8)
func (d *WindowsDriver) applyScaleRatio(imagePath string, scaleRatio int, params models.ScanParams) error {
	if scaleRatio <= 1 {
		return nil // No scaling needed
	}

	// 1. Open and decode image
	file, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}
	file.Close() // Close early to allow overwriting

	// 2. Calculate new dimensions (NAPS2: scaleFactor = 1.0 / scaleRatio)
	bounds := img.Bounds()
	oldWidth := bounds.Dx()
	oldHeight := bounds.Dy()
	newWidth := oldWidth / scaleRatio
	newHeight := oldHeight / scaleRatio

	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	fmt.Printf("  Scaling image: %dx%d -> %dx%d (ratio 1:%d)\n",
		oldWidth, oldHeight, newWidth, newHeight, scaleRatio)

	// 3. Resize using high-quality Lanczos3 interpolation
	scaled := resize.Resize(
		uint(newWidth),
		uint(newHeight),
		img,
		resize.Lanczos3, // High-quality interpolation (NAPS2 uses similar)
	)

	// 4. Save scaled image
	outFile, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Use appropriate format
	if format == "png" || params.MaxQuality {
		return savePNG(scaled, outFile)
	}

	quality := params.JpegQuality
	if quality == 0 {
		quality = models.DefaultJpegQuality // 75
	}
	return saveJPEG(scaled, outFile, quality)
}

// applyCropToPageSize crops or resizes image to match target page size
// Implements NAPS2's crop/stretch to page size feature
func (d *WindowsDriver) applyCropToPageSize(imagePath string, params models.ScanParams) error {
	if !params.CropToPageSize && !params.StretchToPageSize {
		return nil // No processing needed
	}

	// 1. Get target page dimensions in millimeters
	var pageWidthMM, pageHeightMM int
	if params.PageSize != "" && params.PageSize != "Custom" {
		if size, ok := models.PaperSizes[params.PageSize]; ok {
			pageWidthMM = size.Width
			pageHeightMM = size.Height
		}
	} else {
		pageWidthMM = params.PageWidth
		pageHeightMM = params.PageHeight

		// Fallback to legacy fields
		if pageWidthMM == 0 {
			pageWidthMM = params.Width
		}
		if pageHeightMM == 0 {
			pageHeightMM = params.Height
		}
	}

	// Default to A4 if no size specified
	if pageWidthMM == 0 && pageHeightMM == 0 {
		pageWidthMM = 210 // A4
		pageHeightMM = 297
	}

	// 2. Convert to pixels using scan resolution
	resolution := params.Resolution
	if resolution == 0 {
		resolution = 300
	}

	targetWidth := int(float64(pageWidthMM) / 25.4 * float64(resolution))
	targetHeight := int(float64(pageHeightMM) / 25.4 * float64(resolution))

	// 3. Open and decode image
	img, err := imaging.Open(imagePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	bounds := img.Bounds()
	currentWidth := bounds.Dx()
	currentHeight := bounds.Dy()

	// 4. Detect orientation and swap page dimensions if needed (NAPS2 pattern)
	isImageLandscape := currentWidth > currentHeight
	isPageLandscape := targetWidth > targetHeight

	if isImageLandscape != isPageLandscape {
		// Swap target dimensions to match orientation
		targetWidth, targetHeight = targetHeight, targetWidth
		fmt.Printf("  Swapped page dimensions to match orientation: %dx%d\n", targetWidth, targetHeight)
	}

	var processed image.Image

	// 5. Apply transformation
	if params.CropToPageSize {
		// Crop mode: physically crop image to target size
		if currentWidth > targetWidth || currentHeight > targetHeight {
			processed = imaging.CropCenter(img, targetWidth, targetHeight)
			fmt.Printf("  Cropped to page size: %dx%d -> %dx%d\n",
				currentWidth, currentHeight, targetWidth, targetHeight)
		} else {
			processed = img // Image is already smaller than target
			fmt.Printf("  Image smaller than page size, no crop needed\n")
		}
	} else if params.StretchToPageSize {
		// Stretch mode: resize to fit within target size while preserving aspect ratio
		processed = imaging.Fit(img, targetWidth, targetHeight, imaging.Lanczos)
		newBounds := processed.Bounds()
		fmt.Printf("  Resized to fit page: %dx%d -> %dx%d\n",
			currentWidth, currentHeight, newBounds.Dx(), newBounds.Dy())
	} else {
		processed = img
	}

	// 6. Save processed image
	quality := params.JpegQuality
	if quality == 0 {
		quality = models.DefaultJpegQuality
	}

	// Use PNG for MaxQuality, otherwise JPEG
	var saveErr error
	if params.MaxQuality {
		saveErr = imaging.Save(processed, imagePath, imaging.PNGCompressionLevel(png.BestCompression))
	} else {
		saveErr = imaging.Save(processed, imagePath, imaging.JPEGQuality(quality))
	}

	return saveErr
}
