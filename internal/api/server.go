package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/scanserver/scanner-service/internal/scanner"
	"github.com/scanserver/scanner-service/pkg/models"
)

// Server represents the API server
type Server struct {
	router         *gin.Engine
	scannerManager *scanner.Manager
	jobs           map[string]*models.ScanJob
	jobsMutex      sync.RWMutex
	wsHub          *WebSocketHub
}

// NewServer creates a new API server
func NewServer(scannerManager *scanner.Manager, wsHub *WebSocketHub) *Server {
	s := &Server{
		router:         gin.Default(),
		scannerManager: scannerManager,
		jobs:           make(map[string]*models.ScanJob),
		wsHub:          wsHub,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// CORS middleware - 允许跨域访问
	s.router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Scanner endpoints
		v1.GET("/scanners", s.listScanners)
		v1.GET("/scanners/:id", s.getScanner)

		// Scan job endpoints
		v1.POST("/scan", s.createScanJob)
		v1.GET("/jobs", s.listJobs)
		v1.GET("/jobs/:id", s.getJob)
		v1.DELETE("/jobs/:id", s.cancelJob)

		// Batch scan endpoint
		v1.POST("/scan/batch", s.createBatchScan)

		// Scanned files endpoint
		v1.GET("/files/*filepath", s.serveScannedFile)

		// Health check
		v1.GET("/health", s.healthCheck)
	}

	// Serve static files for web UI
	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("./web/templates/*")
	s.router.GET("/", s.serveDashboard)
}

// listScanners returns all available scanners
func (s *Server) listScanners(c *gin.Context) {
	scanners, err := s.scannerManager.ListScanners(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"scanners": scanners})
}

// getScanner returns a specific scanner
func (s *Server) getScanner(c *gin.Context) {
	scannerID := c.Param("id")

	scanner, err := s.scannerManager.GetScanner(c.Request.Context(), scannerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, scanner)
}

// createScanJob creates a new scan job
func (s *Server) createScanJob(c *gin.Context) {
	var req struct {
		ScannerID  string             `json:"scanner_id" binding:"required"`
		Parameters models.ScanParams  `json:"parameters" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create job
	job := &models.ScanJob{
		ID:         models.GenerateUUID(),
		ScannerID:  req.ScannerID,
		Status:     "pending",
		Progress:   0,
		Parameters: req.Parameters,
		Results:    []models.ScanResult{},
		CreatedAt:  time.Now(),
	}

	s.jobsMutex.Lock()
	s.jobs[job.ID] = job
	s.jobsMutex.Unlock()

	// Start scan in background
	go s.executeScanJob(job)

	c.JSON(http.StatusCreated, job)
}

// executeScanJob executes a scan job
func (s *Server) executeScanJob(job *models.ScanJob) {
	ctx := context.Background()

	// Update job status
	s.updateJobStatus(job.ID, "processing", 0)
	s.broadcastJobUpdate(job)

	// Progress callback
	progressCallback := func(progress int) {
		s.updateJobStatus(job.ID, "processing", progress)
		s.broadcastJobUpdate(job)
	}

	// Execute scan
	results, err := s.scannerManager.Scan(ctx, job.ScannerID, job.Parameters, progressCallback)

	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	if err != nil {
		job.Status = "failed"
		job.Error = err.Error()
	} else {
		job.Status = "completed"
		job.Results = results
		job.Progress = 100
	}

	now := time.Now()
	job.CompletedAt = &now

	// Broadcast final status
	s.broadcastJobUpdate(job)
}

// createBatchScan creates a NAPS2-style batch scan
func (s *Server) createBatchScan(c *gin.Context) {
	var req struct {
		ScannerID     string                `json:"scanner_id" binding:"required"`
		Parameters    models.ScanParams     `json:"parameters" binding:"required"`
		BatchSettings models.BatchSettings  `json:"batch_settings" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fill in scan params in batch settings
	req.BatchSettings.ScanParams = req.Parameters

	ctx := context.Background()

	// Create batch scan performer
	performer := scanner.NewBatchScanPerformer(s.scannerManager.GetDriver())

	// Progress callback
	progressCallback := func(progress models.BatchScanProgress) {
		// Broadcast progress via WebSocket
		if s.wsHub != nil {
			msg := models.WebSocketMessage{
				Type:    "batch_scan_progress",
				Payload: progress,
				Time:    time.Now(),
			}
			s.wsHub.Broadcast(msg)
		}
	}

	// Execute batch scan
	scans, err := performer.PerformBatchScan(ctx, req.ScannerID, req.BatchSettings, progressCallback)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Batch scan failed: " + err.Error(),
		})
		return
	}

	// Calculate totals
	totalScans := len(scans)
	totalPages := 0
	for _, scan := range scans {
		totalPages += len(scan)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Batch scan completed successfully",
		"total_scans": totalScans,
		"total_pages": totalPages,
		"scans":       scans,
	})
}

// listJobs returns all jobs
func (s *Server) listJobs(c *gin.Context) {
	s.jobsMutex.RLock()
	defer s.jobsMutex.RUnlock()

	jobs := make([]*models.ScanJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// getJob returns a specific job
func (s *Server) getJob(c *gin.Context) {
	jobID := c.Param("id")

	s.jobsMutex.RLock()
	job, ok := s.jobs[jobID]
	s.jobsMutex.RUnlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// cancelJob cancels a running job
func (s *Server) cancelJob(c *gin.Context) {
	jobID := c.Param("id")

	s.jobsMutex.Lock()
	job, ok := s.jobs[jobID]
	s.jobsMutex.Unlock()

	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	if job.Status != "processing" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job is not running"})
		return
	}

	// Cancel scan
	err := s.scannerManager.CancelScan(c.Request.Context(), job.ScannerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	job.Status = "cancelled"
	now := time.Now()
	job.CompletedAt = &now

	s.broadcastJobUpdate(job)

	c.JSON(http.StatusOK, gin.H{"message": "job cancelled"})
}

// healthCheck returns server health status
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now(),
	})
}

// serveScannedFile serves scanned image files
func (s *Server) serveScannedFile(c *gin.Context) {
	filepath := c.Param("filepath")

	// Security: prevent directory traversal
	if filepath == "" || filepath[0] != '/' {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file path"})
		return
	}

	// Remove leading slash
	filepath = filepath[1:]

	// Serve the file from the scans directory
	c.File(filepath)
}

// serveDashboard serves the web dashboard
func (s *Server) serveDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Scanner Service Dashboard",
	})
}

// updateJobStatus updates job status and progress
func (s *Server) updateJobStatus(jobID string, status string, progress int) {
	s.jobsMutex.Lock()
	defer s.jobsMutex.Unlock()

	if job, ok := s.jobs[jobID]; ok {
		job.Status = status
		job.Progress = progress
	}
}

// broadcastJobUpdate broadcasts job update via WebSocket
func (s *Server) broadcastJobUpdate(job *models.ScanJob) {
	if s.wsHub != nil {
		msg := models.WebSocketMessage{
			Type:    "job_status",
			Payload: job,
			Time:    time.Now(),
		}
		s.wsHub.Broadcast(msg)
	}
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// Router returns the Gin router (useful for testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}

