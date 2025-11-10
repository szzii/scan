package models

import "time"

// Paper sizes in mm (based on NAPS2)
var PaperSizes = map[string]PageDimensions{
	"Letter": {Width: 216, Height: 279},  // 8.5" x 11"
	"Legal":  {Width: 216, Height: 356},  // 8.5" x 14"
	"A4":     {Width: 210, Height: 297},  // ISO A4
	"A3":     {Width: 297, Height: 420},  // ISO A3
	"A5":     {Width: 148, Height: 210},  // ISO A5
	"B4":     {Width: 250, Height: 353},  // ISO B4
	"B5":     {Width: 176, Height: 250},  // ISO B5
	"A6":     {Width: 105, Height: 148},  // ISO A6
}

// PageDimensions represents page size in millimeters
type PageDimensions struct {
	Width  int `json:"width"`  // mm
	Height int `json:"height"` // mm
}

// HorizontalAlign represents horizontal alignment options
const (
	AlignLeft   = "Left"
	AlignCenter = "Center"
	AlignRight  = "Right" // Default
)

// ScaleRatio constants
const (
	Scale1to1 = 1 // No scaling
	Scale1to2 = 2 // 50% (1:2)
	Scale1to4 = 4 // 25% (1:4)
	Scale1to8 = 8 // 12.5% (1:8)
)

// Blank page detection defaults (NAPS2 values)
const (
	DefaultBlankPageWhiteThreshold     = 70 // 0-100
	DefaultBlankPageCoverageThreshold  = 15 // 0-100
	DefaultJpegQuality                 = 75 // 0-100
)

// Scanner represents a scanner device
type Scanner struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Model        string     `json:"model"`
	Manufacturer string     `json:"manufacturer"`
	Status       string     `json:"status"` // idle, scanning, error
	Capabilities Capability `json:"capabilities"`
	LastSeen     time.Time  `json:"last_seen"`
}

// Capability represents scanner capabilities
type Capability struct {
	MaxWidth        int      `json:"max_width"`
	MaxHeight       int      `json:"max_height"`
	Resolutions     []int    `json:"resolutions"`
	ColorModes      []string `json:"color_modes"`      // Color, Grayscale, BlackAndWhite
	DocumentFormats []string `json:"document_formats"` // PDF, JPEG, PNG, TIFF
	FeederEnabled   bool     `json:"feeder_enabled"`
	DuplexEnabled   bool     `json:"duplex_enabled"`
}

// ScanJob represents a scanning job
type ScanJob struct {
	ID          string       `json:"id"`
	ScannerID   string       `json:"scanner_id"`
	Status      string       `json:"status"`   // pending, processing, completed, failed
	Progress    int          `json:"progress"` // 0-100
	Parameters  ScanParams   `json:"parameters"`
	Results     []ScanResult `json:"results"`
	CreatedAt   time.Time    `json:"created_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// ScanParams represents scan parameters (based on NAPS2)
type ScanParams struct {
	// Basic settings
	Resolution int    `json:"resolution"` // DPI
	ColorMode  string `json:"color_mode"` // Color, Grayscale, BlackAndWhite
	Format     string `json:"format"`     // PDF, JPEG, PNG, TIFF

	// Paper source
	UseDuplex bool `json:"use_duplex"`
	UseFeeder bool `json:"use_feeder"`
	PageCount int  `json:"page_count"` // For batch scanning, 0 = unlimited

	// Page size (NAPS2 feature)
	PageSize      string `json:"page_size"`       // Letter, Legal, A4, A3, A5, B4, B5, Custom
	PageWidth     int    `json:"page_width"`      // mm (for custom size)
	PageHeight    int    `json:"page_height"`     // mm (for custom size)
	PageAlign     string `json:"page_align"`      // Left, Center, Right (default: Right)
	WiaOffsetWidth bool  `json:"wia_offset_width"` // Apply horizontal offset

	// Image adjustments
	Brightness int `json:"brightness"` // -1000 to 1000 (WIA scale)
	Contrast   int `json:"contrast"`   // -1000 to 1000 (WIA scale)

	// Scaling and cropping (NAPS2 features)
	ScaleRatio       int  `json:"scale_ratio"`        // 1, 2, 4, 8 (1:1, 1:2, 1:4, 1:8)
	StretchToPageSize bool `json:"stretch_to_page_size"` // Adjust DPI to match page size
	CropToPageSize    bool `json:"crop_to_page_size"`    // Crop image to match page size

	// Image quality (NAPS2 features)
	MaxQuality   bool `json:"max_quality"`   // Lossless quality (overrides Quality)
	JpegQuality  int  `json:"jpeg_quality"`  // 0-100 (default: 75)

	// Blank page detection (NAPS2 features)
	ExcludeBlankPages      bool `json:"exclude_blank_pages"`        // Skip blank pages
	BlankPageWhiteThreshold int `json:"blank_page_white_threshold"` // 0-100 (default: 70)
	BlankPageCoverageThreshold int `json:"blank_page_coverage_threshold"` // 0-100 (default: 15)

	// Advanced options
	AutoDeskew         bool    `json:"auto_deskew"`          // Auto straighten tilted pages
	RotateDegrees      float64 `json:"rotate_degrees"`       // Rotation angle
	FlipDuplexedPages  bool    `json:"flip_duplexed_pages"`  // Flip back side of duplex pages

	// Legacy fields (kept for compatibility)
	Width  int `json:"width"`  // mm (deprecated, use PageWidth)
	Height int `json:"height"` // mm (deprecated, use PageHeight)
}

// ScanResult represents a scanned document
type ScanResult struct {
	PageNumber int    `json:"page_number"`
	FilePath   string `json:"file_path"`
	FileSize   int64  `json:"file_size"`
	Format     string `json:"format"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
}

// WebSocketMessage represents a message sent via WebSocket
type WebSocketMessage struct {
	Type    string      `json:"type"` // job_status, scanner_status, error
	Payload interface{} `json:"payload"`
	Time    time.Time   `json:"time"`
}

// BatchScanType represents the type of batch scanning (NAPS2)
type BatchScanType string

const (
	BatchScanSingle            BatchScanType = "single"              // Single scan
	BatchScanMultipleWithPrompt BatchScanType = "multiple_with_prompt" // Multiple scans with user prompt
	BatchScanMultipleWithDelay  BatchScanType = "multiple_with_delay"  // Multiple scans with delay
)

// BatchOutputType represents how batch scan results are output (NAPS2)
type BatchOutputType string

const (
	BatchOutputLoad         BatchOutputType = "load"          // Load into application (return results)
	BatchOutputSingleFile   BatchOutputType = "single_file"   // Save all pages as single file
	BatchOutputMultipleFiles BatchOutputType = "multiple_files" // Save as multiple files
)

// SaveSeparator represents how to separate multiple files (NAPS2)
type SaveSeparator string

const (
	SaveSeparatorNone       SaveSeparator = "none"        // No separation, all in one file
	SaveSeparatorFilePerScan SaveSeparator = "file_per_scan" // One file per scan
	SaveSeparatorFilePerPage SaveSeparator = "file_per_page" // One file per page
	SaveSeparatorPatchT      SaveSeparator = "patch_t"      // Separate by Patch-T barcode
)

// BatchSettings represents batch scanning configuration (NAPS2)
type BatchSettings struct {
	ProfileDisplayName string `json:"profile_display_name"` // Scan profile name

	ScanType             BatchScanType   `json:"scan_type"`              // Single, MultipleWithPrompt, MultipleWithDelay
	ScanCount            int             `json:"scan_count"`             // Number of scans (for MultipleWithDelay)
	ScanIntervalSeconds  float64         `json:"scan_interval_seconds"`  // Interval between scans (for MultipleWithDelay)

	OutputType           BatchOutputType `json:"output_type"`            // Load, SingleFile, MultipleFiles
	SaveSeparator        SaveSeparator   `json:"save_separator"`         // How to separate multiple files
	SavePath             string          `json:"save_path"`              // Path pattern for saving files

	// Scan parameters
	ScanParams           ScanParams      `json:"scan_params"`            // Scan parameters to use
}

// BatchScanProgress represents progress during batch scanning
type BatchScanProgress struct {
	Stage           string  `json:"stage"`            // "scanning" or "saving"
	CurrentScan     int     `json:"current_scan"`     // Current scan number (1-based)
	TotalScans      int     `json:"total_scans"`      // Total number of scans
	CurrentPage     int     `json:"current_page"`     // Current page number (1-based)
	TotalPages      int     `json:"total_pages"`      // Total pages scanned
	Message         string  `json:"message"`          // Status message
	PercentComplete int     `json:"percent_complete"` // 0-100
}
