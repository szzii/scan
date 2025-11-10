package escl

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scanserver/scanner-service/internal/scanner"
)

// eSCL (eScan over HTTP) protocol implementation
// This is the Apple AirPrint scanning protocol

// ScannerCapabilities represents eSCL scanner capabilities XML
type ScannerCapabilities struct {
	XMLName             xml.Name `xml:"scan:ScannerCapabilities"`
	Xmlns               string   `xml:"xmlns:scan,attr"`
	XmlnsPwg            string   `xml:"xmlns:pwg,attr"`
	Version             string   `xml:"scan:Version"`
	MakeAndModel        string   `xml:"scan:MakeAndModel"`
	Manufacturer        string   `xml:"scan:Manufacturer"`
	SerialNumber        string   `xml:"scan:SerialNumber"`
	UUID                string   `xml:"scan:UUID"`
	AdminURI            string   `xml:"scan:AdminURI"`
	IconURI             string   `xml:"scan:IconURI"`
	Platen              *Platen  `xml:"scan:Platen,omitempty"`
	Adf                 *Adf     `xml:"scan:Adf,omitempty"`
}

type Platen struct {
	PlatenInputCaps PlatenInputCaps `xml:"scan:PlatenInputCaps"`
}

type PlatenInputCaps struct {
	MinWidth          int              `xml:"scan:MinWidth"`
	MaxWidth          int              `xml:"scan:MaxWidth"`
	MinHeight         int              `xml:"scan:MinHeight"`
	MaxHeight         int              `xml:"scan:MaxHeight"`
	MaxScanRegions    int              `xml:"scan:MaxScanRegions"`
	SettingProfiles   SettingProfiles  `xml:"scan:SettingProfiles"`
	SupportedIntents  SupportedIntents `xml:"scan:SupportedIntents"`
	MaxOpticalXResolution int          `xml:"scan:MaxOpticalXResolution"`
	MaxOpticalYResolution int          `xml:"scan:MaxOpticalYResolution"`
	RiskyLeftMargin   int              `xml:"scan:RiskyLeftMargin"`
	RiskyRightMargin  int              `xml:"scan:RiskyRightMargin"`
	RiskyTopMargin    int              `xml:"scan:RiskyTopMargin"`
	RiskyBottomMargin int              `xml:"scan:RiskyBottomMargin"`
}

type Adf struct {
	AdfSimplexInputCaps *AdfInputCaps `xml:"scan:AdfSimplexInputCaps,omitempty"`
	AdfDuplexInputCaps  *AdfInputCaps `xml:"scan:AdfDuplexInputCaps,omitempty"`
}

type AdfInputCaps struct {
	MinWidth          int              `xml:"scan:MinWidth"`
	MaxWidth          int              `xml:"scan:MaxWidth"`
	MinHeight         int              `xml:"scan:MinHeight"`
	MaxHeight         int              `xml:"scan:MaxHeight"`
	MaxScanRegions    int              `xml:"scan:MaxScanRegions"`
	SettingProfiles   SettingProfiles  `xml:"scan:SettingProfiles"`
	SupportedIntents  SupportedIntents `xml:"scan:SupportedIntents"`
	MaxOpticalXResolution int          `xml:"scan:MaxOpticalXResolution"`
	MaxOpticalYResolution int          `xml:"scan:MaxOpticalYResolution"`
}

type SettingProfiles struct {
	SettingProfile []SettingProfile `xml:"scan:SettingProfile"`
}

type SettingProfile struct {
	ColorModes       ColorModes       `xml:"scan:ColorModes"`
	DocumentFormats  DocumentFormats  `xml:"scan:DocumentFormats"`
	SupportedResolutions SupportedResolutions `xml:"scan:SupportedResolutions"`
}

type ColorModes struct {
	ColorMode []string `xml:"scan:ColorMode"`
}

type DocumentFormats struct {
	DocumentFormat []string `xml:"pwg:DocumentFormat"`
}

type SupportedResolutions struct {
	DiscreteResolutions DiscreteResolutions `xml:"scan:DiscreteResolutions"`
}

type DiscreteResolutions struct {
	DiscreteResolution []DiscreteResolution `xml:"scan:DiscreteResolution"`
}

type DiscreteResolution struct {
	XResolution int `xml:"scan:XResolution"`
	YResolution int `xml:"scan:YResolution"`
}

type SupportedIntents struct {
	Intent []string `xml:"scan:Intent"`
}

// ScannerStatus represents eSCL scanner status XML
type ScannerStatus struct {
	XMLName xml.Name `xml:"scan:ScannerStatus"`
	Xmlns   string   `xml:"xmlns:scan,attr"`
	XmlnsPwg string  `xml:"xmlns:pwg,attr"`
	Version string   `xml:"scan:Version"`
	State   string   `xml:"pwg:State"`
	StateReasons StateReasons `xml:"pwg:StateReasons"`
}

type StateReasons struct {
	StateReason []string `xml:"pwg:StateReason"`
}

// ESCLServer handles eSCL protocol requests
type ESCLServer struct {
	scannerManager *scanner.Manager
}

// NewESCLServer creates a new eSCL server
func NewESCLServer(scannerManager *scanner.Manager) *ESCLServer {
	return &ESCLServer{
		scannerManager: scannerManager,
	}
}

// RegisterRoutes registers eSCL routes
func (s *ESCLServer) RegisterRoutes(router *gin.Engine) {
	escl := router.Group("/eSCL")
	{
		escl.GET("/ScannerCapabilities", s.getScannerCapabilities)
		escl.GET("/ScannerStatus", s.getScannerStatus)
		escl.POST("/ScanJobs", s.createScanJob)
		escl.GET("/ScanJobs/:jobId/NextDocument", s.getNextDocument)
		escl.DELETE("/ScanJobs/:jobId", s.deleteScanJob)
	}
}

// getScannerCapabilities returns scanner capabilities in eSCL format
func (s *ESCLServer) getScannerCapabilities(c *gin.Context) {
	scanners, err := s.scannerManager.ListScanners(c.Request.Context())
	if err != nil || len(scanners) == 0 {
		c.XML(http.StatusNotFound, gin.H{"error": "no scanner available"})
		return
	}

	scanner := scanners[0] // Use first available scanner

	caps := ScannerCapabilities{
		Xmlns:        "http://schemas.hp.com/imaging/escl/2011/05/03",
		XmlnsPwg:     "http://www.pwg.org/schemas/2010/12/sm",
		Version:      "2.6",
		MakeAndModel: fmt.Sprintf("%s %s", scanner.Manufacturer, scanner.Model),
		Manufacturer: scanner.Manufacturer,
		SerialNumber: scanner.ID,
		UUID:         fmt.Sprintf("urn:uuid:%s", scanner.ID),
		AdminURI:     "http://localhost:8080/",
		IconURI:      "http://localhost:8080/static/icon.png",
		Platen: &Platen{
			PlatenInputCaps: PlatenInputCaps{
				MinWidth:  1,
				MaxWidth:  scanner.Capabilities.MaxWidth,
				MinHeight: 1,
				MaxHeight: scanner.Capabilities.MaxHeight,
				MaxScanRegions: 1,
				SettingProfiles: SettingProfiles{
					SettingProfile: []SettingProfile{
						{
							ColorModes: ColorModes{
								ColorMode: scanner.Capabilities.ColorModes,
							},
							DocumentFormats: DocumentFormats{
								DocumentFormat: []string{"image/jpeg", "application/pdf"},
							},
							SupportedResolutions: s.buildResolutions(scanner.Capabilities.Resolutions),
						},
					},
				},
				SupportedIntents: SupportedIntents{
					Intent: []string{"Document", "Photo", "Preview"},
				},
				MaxOpticalXResolution: 1200,
				MaxOpticalYResolution: 1200,
				RiskyLeftMargin:       0,
				RiskyRightMargin:      0,
				RiskyTopMargin:        0,
				RiskyBottomMargin:     0,
			},
		},
	}

	if scanner.Capabilities.FeederEnabled {
		caps.Adf = &Adf{
			AdfSimplexInputCaps: &AdfInputCaps{
				MinWidth:  1,
				MaxWidth:  scanner.Capabilities.MaxWidth,
				MinHeight: 1,
				MaxHeight: scanner.Capabilities.MaxHeight,
				MaxScanRegions: 1,
				SettingProfiles: SettingProfiles{
					SettingProfile: []SettingProfile{
						{
							ColorModes: ColorModes{
								ColorMode: scanner.Capabilities.ColorModes,
							},
							DocumentFormats: DocumentFormats{
								DocumentFormat: []string{"image/jpeg", "application/pdf"},
							},
							SupportedResolutions: s.buildResolutions(scanner.Capabilities.Resolutions),
						},
					},
				},
				SupportedIntents: SupportedIntents{
					Intent: []string{"Document"},
				},
				MaxOpticalXResolution: 1200,
				MaxOpticalYResolution: 1200,
			},
		}

		if scanner.Capabilities.DuplexEnabled {
			caps.Adf.AdfDuplexInputCaps = caps.Adf.AdfSimplexInputCaps
		}
	}

	c.XML(http.StatusOK, caps)
}

// getScannerStatus returns scanner status in eSCL format
func (s *ESCLServer) getScannerStatus(c *gin.Context) {
	status := ScannerStatus{
		Xmlns:    "http://schemas.hp.com/imaging/escl/2011/05/03",
		XmlnsPwg: "http://www.pwg.org/schemas/2010/12/sm",
		Version:  "2.6",
		State:    "Idle",
		StateReasons: StateReasons{
			StateReason: []string{"None"},
		},
	}

	c.XML(http.StatusOK, status)
}

// createScanJob creates a new scan job via eSCL
func (s *ESCLServer) createScanJob(c *gin.Context) {
	// Parse eSCL scan settings XML from request body
	// For simplicity, using default settings

	jobID := fmt.Sprintf("escl-job-%d", time.Now().Unix())

	// Return job location
	c.Header("Location", fmt.Sprintf("/eSCL/ScanJobs/%s", jobID))
	c.Status(http.StatusCreated)
}

// getNextDocument retrieves the next scanned document
func (s *ESCLServer) getNextDocument(c *gin.Context) {
	jobID := c.Param("jobId")

	// In a real implementation, would return actual scanned image data
	// For now, return mock response

	c.Header("Content-Type", "image/jpeg")
	c.Status(http.StatusOK)
	// Would send actual JPEG data here
	c.String(http.StatusOK, fmt.Sprintf("Mock scan data for job %s", jobID))
}

// deleteScanJob deletes a scan job
func (s *ESCLServer) deleteScanJob(c *gin.Context) {
	jobID := c.Param("jobId")

	// Delete job
	_ = jobID

	c.Status(http.StatusOK)
}

// buildResolutions converts resolution array to eSCL format
func (s *ESCLServer) buildResolutions(resolutions []int) SupportedResolutions {
	var discreteResolutions []DiscreteResolution

	for _, res := range resolutions {
		discreteResolutions = append(discreteResolutions, DiscreteResolution{
			XResolution: res,
			YResolution: res,
		})
	}

	return SupportedResolutions{
		DiscreteResolutions: DiscreteResolutions{
			DiscreteResolution: discreteResolutions,
		},
	}
}
