# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Scanner Service is a cross-platform document scanning service (Go 1.21) that replicates NAPS2's functionality. It provides RESTful APIs, WebSocket real-time updates, eSCL protocol support (Apple AirPrint), batch scanning, and automatic lid-close scanning.

## Common Commands

```bash
# Development
make run              # Run the application
make run-dev          # Run with localhost:8080
make build            # Build for current platform

# Testing & Quality
make test             # Run tests with coverage
make test-coverage    # Run tests and show HTML coverage report
make lint             # Run golangci-lint (requires installation)
make vet              # Run go vet
make fmt              # Format code

# Building
make build-all        # Build for Windows, Linux, macOS
make build-windows    # Windows AMD64 only
make build-linux      # Linux AMD64 only
make build-macos      # macOS Intel + ARM64

# Docker
make docker-build     # Build Docker image
make docker-run       # Run container (port 8080)

# Dependencies
make install-deps     # Install and tidy Go dependencies
make deps-update      # Update all dependencies
make setup            # Full dev environment setup
```

## Architecture

### Platform-Specific Driver Pattern

The scanner abstraction uses build tags for platform-specific implementations:

- `internal/scanner/interface.go` - Defines `ScannerDriver` interface and `Manager` wrapper
- `internal/scanner/driver_windows.go` - Windows WIA implementation (most mature)
- `internal/scanner/driver_linux.go` - Linux SANE implementation (template)
- `internal/scanner/driver_darwin.go` - macOS ImageCaptureCore implementation (template)

The `Manager` struct wraps the driver and is instantiated via `NewManager()` which calls `newPlatformDriver()` - selected at compile time by build tags.

### Core Components

```
cmd/scanserver/         # Main entry point - flag parsing, server startup, graceful shutdown
internal/
  api/                  # Gin HTTP handlers + WebSocket hub
  config/               # Viper-based configuration
  escl/                 # eSCL (AirPrint) protocol XML endpoints
  scanner/              # Driver abstraction, batch scanning, auto-scan
pkg/models/             # Shared data structures (Scanner, ScanParams, ScanJob, etc.)
web/                    # HTML templates + static assets for dashboard
sdk/javascript/         # Client SDK for browser/Node.js
```

### Key Patterns

- **WebSocket Hub** (`internal/api/websocket.go`): Manages client connections and broadcasts job/scanner status updates
- **Batch Scanning** (`internal/scanner/batch.go`): NAPS2-compatible workflows (single, multiple with delay/prompt, separators)
- **Auto-Scan** (`internal/scanner/autoscan.go`): Monitors lid-close events and triggers automatic scanning
- **Progress Callbacks**: All scan operations accept `progressCallback func(int)` for real-time progress reporting

### NAPS2 Compatibility

This codebase explicitly implements NAPS2 features including:
- WIA transfer loops and property handling (30+ WIA property constants)
- Paper sizing (8 presets + custom dimensions)
- Image processing (scaling, cropping, auto-deskew, rotation)
- Quality control (JPEG quality 0-100, max quality PNG)
- Blank page detection (YUV-based thresholds)
- Batch scanning workflows with separators

### API Endpoints

- `GET/POST /api/v1/scanners`, `/api/v1/scan`, `/api/v1/scan/batch`, `/api/v1/jobs`
- WebSocket: `ws://localhost:8080/ws`
- eSCL: `/eSCL/ScannerCapabilities`, `/eSCL/ScannerStatus`
- Dashboard: `GET /`

## Configuration

Environment variables use prefix `SCANNER_`. YAML config example at `config.example.yaml`. Key sections: server, scanner defaults, storage, autoscan.

## Platform Requirements

- **Windows**: WIA (built-in)
- **Linux**: `apt install sane-utils libsane-dev`
- **macOS**: ImageCaptureCore (built-in)
