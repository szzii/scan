package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/scanserver/scanner-service/internal/api"
	"github.com/scanserver/scanner-service/internal/config"
	"github.com/scanserver/scanner-service/internal/escl"
	"github.com/scanserver/scanner-service/internal/scanner"
	"github.com/scanserver/scanner-service/pkg/models"
)

var (
	configFile = flag.String("config", "", "Path to configuration file")
	host       = flag.String("host", "0.0.0.0", "Server host")
	port       = flag.Int("port", 8080, "Server port")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("Starting Scanner Service...")
	log.Printf("Version: 1.0.0")
	log.Printf("Platform: %s", getPlatform())

	// Initialize scanner manager
	scannerManager, err := scanner.NewManager()
	if err != nil {
		log.Fatalf("Failed to initialize scanner manager: %v", err)
	}
	defer scannerManager.Close()

	// Create WebSocket hub
	wsHub := api.NewWebSocketHub()
	go wsHub.Run()

	// Create API server
	apiServer := api.NewServer(scannerManager, wsHub)
	apiServer.AddWebSocketRoute()

	// Create eSCL server if enabled
	if cfg.Server.ESCLEnabled {
		esclServer := escl.NewESCLServer(scannerManager)
		esclServer.RegisterRoutes(apiServer.Router())
		log.Println("eSCL protocol support enabled")
	}

	// Initialize auto-scan if enabled
	var autoScanManager *scanner.AutoScanManager
	if cfg.AutoScan.Enabled {
		autoScanManager = scanner.NewAutoScanManager(
			scannerManager,
			&cfg.AutoScan,
			func(job *models.ScanJob) {
				// Broadcast job updates via WebSocket
				msg := models.WebSocketMessage{
					Type:    "job_status",
					Payload: job,
				}
				wsHub.Broadcast(msg)
			},
		)

		err = autoScanManager.Start()
		if err != nil {
			log.Printf("Warning: Failed to start auto-scan: %v", err)
		} else {
			log.Println("Auto-scan (lid close detection) enabled")
		}
		defer autoScanManager.Stop()
	}

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(cfg.Storage.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on http://%s", addr)
	log.Printf("Web dashboard: http://%s/", addr)
	log.Printf("API endpoint: http://%s/api/v1", addr)
	log.Printf("WebSocket: ws://%s/ws", addr)

	if cfg.Server.ESCLEnabled {
		log.Printf("eSCL endpoint: http://%s/eSCL", addr)
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		if autoScanManager != nil {
			autoScanManager.Stop()
		}
		scannerManager.Close()
		os.Exit(0)
	}()

	// Run server
	if err := apiServer.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(*configFile)
	if err != nil && *configFile != "" {
		return nil, err
	}

	// Override with command-line flags
	if *host != "0.0.0.0" {
		cfg.Server.Host = *host
	}
	if *port != 8080 {
		cfg.Server.Port = *port
	}

	return cfg, nil
}

func getPlatform() string {
	switch {
	case os.Getenv("OS") == "Windows_NT":
		return "Windows"
	case fileExists("/System/Library/CoreServices/SystemVersion.plist"):
		return "macOS"
	default:
		return "Linux"
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
