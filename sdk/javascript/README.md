# Scanner Service JavaScript SDK

JavaScript/TypeScript SDK for the Scanner Service - A cross-platform document scanning service.

## Features

- List and manage scanners
- Create scan jobs with customizable parameters
- Batch scanning support
- Real-time progress updates via WebSocket
- Works in both Node.js and Browser environments
- Automatic WebSocket reconnection
- Promise-based API

## Installation

### For Node.js

```bash
npm install node-fetch ws
```

### For Browser

Simply include the SDK script:

```html
<script src="path/to/scanner-client.js"></script>
```

## Quick Start

### Node.js Example

```javascript
const ScannerClient = require('./src/scanner-client');

// Polyfills for Node.js
global.fetch = require('node-fetch');
global.WebSocket = require('ws');

const client = new ScannerClient('http://localhost:8080');

async function scan() {
    // List scanners
    const scanners = await client.listScanners();
    console.log('Scanners:', scanners);

    // Connect WebSocket for real-time updates
    await client.connectWebSocket();

    // Listen for job updates
    client.on('job_status', (job) => {
        console.log(`Job ${job.id}: ${job.status} (${job.progress}%)`);
    });

    // Create a scan job
    const job = await client.createScan(scanners[0].id, {
        resolution: 300,
        color_mode: 'Color',
        format: 'PDF',
        page_count: 1
    });

    console.log('Scan started:', job.id);
}

scan().catch(console.error);
```

### Browser Example

```html
<!DOCTYPE html>
<html>
<head>
    <script src="scanner-client.js"></script>
</head>
<body>
    <button onclick="scan()">Start Scan</button>

    <script>
        const client = new ScannerClient('http://localhost:8080');

        async function scan() {
            // Connect WebSocket
            await client.connectWebSocket();

            // Listen for updates
            client.on('job_status', (job) => {
                console.log(`Progress: ${job.progress}%`);
            });

            // List scanners
            const scanners = await client.listScanners();

            // Start scan
            const job = await client.createScan(scanners[0].id, {
                resolution: 300,
                color_mode: 'Color',
                format: 'PDF'
            });

            alert(`Scan started: ${job.id}`);
        }
    </script>
</body>
</html>
```

## API Reference

### Constructor

```javascript
const client = new ScannerClient(baseURL, options)
```

**Parameters:**
- `baseURL` (string): The base URL of the scanner service (e.g., 'http://localhost:8080')
- `options` (object, optional):
  - `autoReconnect` (boolean): Auto-reconnect WebSocket on disconnect (default: true)
  - `reconnectInterval` (number): WebSocket reconnect interval in ms (default: 3000)

### Methods

#### Scanner Management

**`listScanners()`**

Returns a list of all available scanners.

```javascript
const scanners = await client.listScanners();
```

**`getScanner(scannerId)`**

Get details of a specific scanner.

```javascript
const scanner = await client.getScanner('scanner-001');
```

#### Scan Jobs

**`createScan(scannerId, parameters)`**

Create a new scan job.

```javascript
const job = await client.createScan('scanner-001', {
    resolution: 300,        // DPI
    color_mode: 'Color',    // 'Color', 'Grayscale', 'BlackAndWhite'
    format: 'PDF',          // 'PDF', 'JPEG', 'PNG', 'TIFF'
    width: 210,             // mm (A4 width)
    height: 297,            // mm (A4 height)
    page_count: 1,          // Number of pages
    use_duplex: false,      // Use duplex scanning
    use_feeder: false       // Use document feeder
});
```

**`createBatchScan(scannerId, parameters, batchCount)`**

Create multiple scan jobs.

```javascript
const result = await client.createBatchScan('scanner-001', {
    resolution: 300,
    color_mode: 'Color',
    format: 'PDF'
}, 5);  // Create 5 scan jobs

console.log('Job IDs:', result.job_ids);
```

**`listJobs()`**

Get a list of all scan jobs.

```javascript
const jobs = await client.listJobs();
```

**`getJob(jobId)`**

Get details of a specific job.

```javascript
const job = await client.getJob('job-12345');
```

**`cancelJob(jobId)`**

Cancel a running scan job.

```javascript
await client.cancelJob('job-12345');
```

#### WebSocket

**`connectWebSocket()`**

Connect to WebSocket for real-time updates.

```javascript
await client.connectWebSocket();
```

**`disconnectWebSocket()`**

Disconnect WebSocket.

```javascript
client.disconnectWebSocket();
```

**`on(eventType, callback)`**

Register an event listener.

```javascript
client.on('job_status', (job) => {
    console.log(`Job update:`, job);
});
```

**Event Types:**
- `job_status`: Job status updates
- `scanner_status`: Scanner status updates
- `connected`: WebSocket connected
- `disconnected`: WebSocket disconnected
- `error`: WebSocket error

**`off(eventType, callback)`**

Remove an event listener.

```javascript
client.off('job_status', myCallback);
```

**`removeAllListeners(eventType)`**

Remove all listeners for an event type.

```javascript
client.removeAllListeners('job_status');
```

#### Utility

**`healthCheck()`**

Check if the service is healthy.

```javascript
const health = await client.healthCheck();
console.log('Status:', health.status);
```

## Examples

See the `examples/` directory for complete examples:

- **`node-example.js`**: Complete Node.js example showing all features
- **`browser-example.html`**: Interactive browser example with UI

### Running Examples

```bash
# Node.js example
npm install
npm run example:node

# Browser example
# Open examples/browser-example.html in your browser
```

## Scan Parameters

### Resolution

Common DPI values:
- `150`: Draft quality
- `300`: Standard quality (recommended)
- `600`: High quality
- `1200`: Very high quality

### Color Modes

- `Color`: Full color scanning
- `Grayscale`: Grayscale scanning
- `BlackAndWhite`: Binary (black and white) scanning

### Output Formats

- `PDF`: Portable Document Format
- `JPEG`: JPEG image
- `PNG`: PNG image
- `TIFF`: TIFF image

## Error Handling

All methods return Promises and should be wrapped in try-catch blocks:

```javascript
try {
    const scanners = await client.listScanners();
} catch (error) {
    console.error('Failed to list scanners:', error.message);
}
```

## WebSocket Events

The SDK automatically manages WebSocket connections and provides real-time updates:

```javascript
// Connect
await client.connectWebSocket();

// Listen for job updates
client.on('job_status', (job) => {
    if (job.status === 'processing') {
        console.log(`Progress: ${job.progress}%`);
    } else if (job.status === 'completed') {
        console.log('Scan completed!');
        console.log('Results:', job.results);
    } else if (job.status === 'failed') {
        console.error('Scan failed:', job.error);
    }
});

// Listen for connection events
client.on('disconnected', () => {
    console.log('WebSocket disconnected');
});

client.on('connected', () => {
    console.log('WebSocket connected');
});
```

## Browser Compatibility

- Modern browsers with fetch API support
- WebSocket support required
- ES6+ features used

## Node.js Compatibility

- Node.js 14.0.0 or higher
- Requires `node-fetch` and `ws` packages for Node.js < 18

## License

MIT
