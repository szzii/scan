# Scanner Service

A cross-platform document scanning service with support for Windows, Linux, and macOS. Features RESTful API, WebSocket real-time updates, eSCL protocol support, batch scanning, and automatic lid-close scanning.

## Features

### Core Features
- **Cross-platform support**: Windows (WIA), Linux (SANE), macOS (ImageCaptureCore)
- **RESTful HTTP API**: Easy integration with any application
- **WebSocket support**: Real-time scan progress updates
- **eSCL protocol**: Apple AirPrint scanning compatibility
- **Web dashboard**: Modern web interface for managing scans
- **Batch scanning**: Scan multiple documents in sequence
- **Auto-scan**: Automatic scanning when scanner lid closes

### Scanning Capabilities
- Multiple resolution support (100-1200 DPI)
- Color modes: Color, Grayscale, Black & White
- Output formats: PDF, JPEG, PNG, TIFF
- Duplex scanning support
- Document feeder support
- Customizable scan parameters

## Quick Start

### Prerequisites

- Go 1.21 or higher (for building from source)
- Scanner connected and drivers installed
- Scanner libraries:
  - **Windows**: WIA (built-in)
  - **Linux**: SANE (`apt install sane-utils libsane-dev`)
  - **macOS**: ImageCaptureCore (built-in)

### Installation

#### Option 1: Download Pre-built Binary

Download the latest release for your platform from the releases page and extract:

```bash
# Linux/macOS
tar -xzf scanserver-linux-amd64-v1.0.0.tar.gz
cd scanserver-linux-amd64

# Windows
# Extract scanserver-windows-amd64-v1.0.0.zip
cd scanserver-windows-amd64
```

#### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/scanserver/scanner-service.git
cd scanner-service

# Install dependencies
go mod download

# Build for current platform
make build

# Or build for all platforms
make build-all
```

### Running the Service

```bash
# Run with default settings
./scanserver

# Run with custom configuration
./scanserver -config config.yaml

# Run with custom host/port
./scanserver -host 0.0.0.0 -port 8080
```

The service will start on `http://localhost:8080` by default.

### Access the Web Dashboard

Open your browser and navigate to:
```
http://localhost:8080
```

## Configuration

Copy the example configuration file:

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` to customize settings:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  escl_enabled: true

scanner:
  default_resolution: 300
  default_color_mode: "Color"
  default_format: "PDF"

storage:
  output_dir: "./scans"
  cleanup_enabled: true
  retention_days: 30

autoscan:
  enabled: false
  lid_close_delay: 2
```

See `config.example.yaml` for all available options.

## API Documentation

### Base URL

```
http://localhost:8080/api/v1
```

### Endpoints

#### List Scanners

```bash
GET /api/v1/scanners

# Example
curl http://localhost:8080/api/v1/scanners
```

Response:
```json
{
  "scanners": [
    {
      "id": "scanner-001",
      "name": "HP LaserJet Scanner",
      "manufacturer": "HP",
      "model": "LaserJet Pro MFP M428fdw",
      "status": "idle",
      "capabilities": {
        "max_width": 8500,
        "max_height": 11700,
        "resolutions": [100, 150, 200, 300, 600, 1200],
        "color_modes": ["Color", "Grayscale", "BlackAndWhite"],
        "document_formats": ["PDF", "JPEG", "PNG", "TIFF"],
        "feeder_enabled": true,
        "duplex_enabled": true
      }
    }
  ]
}
```

#### Create Scan Job

```bash
POST /api/v1/scan
Content-Type: application/json

{
  "scanner_id": "scanner-001",
  "parameters": {
    "resolution": 300,
    "color_mode": "Color",
    "format": "PDF",
    "width": 210,
    "height": 297,
    "page_count": 1
  }
}
```

Example:
```bash
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "scanner_id": "scanner-001",
    "parameters": {
      "resolution": 300,
      "color_mode": "Color",
      "format": "PDF",
      "page_count": 1
    }
  }'
```

#### Create Batch Scan

```bash
POST /api/v1/scan/batch
Content-Type: application/json

{
  "scanner_id": "scanner-001",
  "parameters": {
    "resolution": 300,
    "color_mode": "Color",
    "format": "PDF"
  },
  "batch_count": 5
}
```

#### Get Job Status

```bash
GET /api/v1/jobs/{job_id}

# Example
curl http://localhost:8080/api/v1/jobs/abc-123
```

#### List All Jobs

```bash
GET /api/v1/jobs

# Example
curl http://localhost:8080/api/v1/jobs
```

#### Cancel Job

```bash
DELETE /api/v1/jobs/{job_id}

# Example
curl -X DELETE http://localhost:8080/api/v1/jobs/abc-123
```

### WebSocket

Connect to WebSocket for real-time updates:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Update:', message);
};
```

WebSocket message types:
- `job_status`: Job progress and status updates
- `scanner_status`: Scanner availability changes

### eSCL Protocol

The service supports eSCL (eScan over HTTP) for AirPrint scanning:

```
http://localhost:8080/eSCL/ScannerCapabilities
http://localhost:8080/eSCL/ScannerStatus
```

## JavaScript SDK

A JavaScript SDK is provided for easy integration:

```javascript
const ScannerClient = require('./sdk/javascript/src/scanner-client');

const client = new ScannerClient('http://localhost:8080');

// List scanners
const scanners = await client.listScanners();

// Connect WebSocket
await client.connectWebSocket();

// Listen for updates
client.on('job_status', (job) => {
  console.log(`Progress: ${job.progress}%`);
});

// Start scan
const job = await client.createScan(scanners[0].id, {
  resolution: 300,
  color_mode: 'Color',
  format: 'PDF'
});
```

See `sdk/javascript/README.md` for full SDK documentation.

## Development

### Project Structure

```
scanner-service/
├── cmd/
│   └── scanserver/        # Main application
├── internal/
│   ├── api/               # HTTP API handlers
│   ├── config/            # Configuration management
│   ├── escl/              # eSCL protocol implementation
│   ├── scanner/           # Scanner driver abstraction
│   └── websocket/         # WebSocket handlers
├── pkg/
│   └── models/            # Data models
├── web/
│   ├── static/            # Static assets
│   └── templates/         # HTML templates
├── sdk/
│   └── javascript/        # JavaScript SDK
├── scripts/
│   ├── build.sh           # Unix build script
│   └── build.ps1          # Windows build script
├── go.mod
├── Makefile
└── README.md
```

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platform
make build-windows
make build-linux
make build-macos

# Run tests
make test

# Format code
make fmt
```

### Make Commands

```bash
make help              # Show all available commands
make build             # Build for current platform
make build-all         # Build for all platforms
make run               # Run the application
make test              # Run tests
make clean             # Clean build artifacts
make install-deps      # Install Go dependencies
```

## Platform-Specific Notes

### Windows

- Uses Windows Image Acquisition (WIA) API
- Requires scanner drivers to be installed
- Run as administrator for some scanners

### Linux

- Requires SANE libraries:
  ```bash
  sudo apt-get install sane-utils libsane-dev
  ```
- Check scanner is detected:
  ```bash
  scanimage -L
  ```
- Configure SANE if needed: `/etc/sane.d/`

### macOS

- Uses ImageCaptureCore framework
- Scanner should appear in System Preferences
- May require permission to access scanner

## Auto-Scan Feature

Enable automatic scanning when the scanner lid closes:

1. Edit `config.yaml`:
   ```yaml
   autoscan:
     enabled: true
     lid_close_delay: 2
     default_params:
       resolution: 300
       color_mode: "Color"
       format: "PDF"
   ```

2. Restart the service

3. Close the scanner lid - scanning will start automatically after the configured delay

## Troubleshooting

### Scanner Not Detected

**Windows:**
- Check Device Manager for scanner
- Reinstall scanner drivers
- Try running as administrator

**Linux:**
- Run `scanimage -L` to list scanners
- Check SANE configuration: `/etc/sane.d/`
- Check permissions: user must be in `scanner` group

**macOS:**
- Check System Preferences → Printers & Scanners
- Restart the scanner service

### Port Already in Use

Change the port in configuration:
```bash
./scanserver -port 8081
```

### WebSocket Connection Failed

- Check firewall settings
- Ensure WebSocket upgrade is allowed
- Check browser console for errors

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues, questions, or feature requests, please open an issue on GitHub.

## Roadmap

- [ ] OCR support for scanned documents
- [ ] Cloud storage integration (S3, Google Drive, Dropbox)
- [ ] Email delivery of scanned documents
- [ ] Mobile app for remote scanning
- [ ] Docker support
- [ ] Multi-user authentication
- [ ] Scan templates and presets
- [ ] Network scanner discovery

## Credits

Built with:
- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [Viper](https://github.com/spf13/viper) - Configuration management
