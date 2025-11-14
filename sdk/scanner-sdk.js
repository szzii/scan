/**
 * Scanner Service JavaScript SDK
 * Version: 1.0.0
 *
 * A complete JavaScript SDK for interacting with the Scanner Service API
 */

class ScannerClient {
    /**
     * Initialize Scanner Client
     * @param {string} baseUrl - Base URL of scanner service (e.g., "http://localhost:8080")
     * @param {Object} options - Configuration options
     * @param {boolean} options.autoConnect - Automatically connect WebSocket (default: true)
     * @param {number} options.reconnectInterval - WebSocket reconnect interval in ms (default: 3000)
     */
    constructor(baseUrl, options = {}) {
        this.baseUrl = baseUrl.replace(/\/$/, ''); // Remove trailing slash
        this.apiBase = `${this.baseUrl}/api/v1`;
        this.ws = null;
        this.wsUrl = baseUrl.replace('http://', 'ws://').replace('https://', 'wss://') + '/ws';

        this.options = {
            autoConnect: options.autoConnect !== false,
            reconnectInterval: options.reconnectInterval || 3000
        };

        this.listeners = {
            job_status: [],
            batch_scan_progress: [],
            connected: [],
            disconnected: [],
            error: []
        };

        if (this.options.autoConnect) {
            this.connectWebSocket();
        }
    }

    // ==================== WebSocket Methods ====================

    /**
     * Connect to WebSocket for real-time updates
     */
    connectWebSocket() {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            console.log('WebSocket already connected');
            return;
        }

        try {
            this.ws = new WebSocket(this.wsUrl);

            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this._emit('connected');
            };

            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this._emit(message.type, message.payload);
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };

            this.ws.onclose = () => {
                console.log('WebSocket disconnected');
                this._emit('disconnected');

                // Auto reconnect
                setTimeout(() => {
                    console.log('Reconnecting WebSocket...');
                    this.connectWebSocket();
                }, this.options.reconnectInterval);
            };

            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this._emit('error', error);
            };
        } catch (error) {
            console.error('Failed to connect WebSocket:', error);
            this._emit('error', error);
        }
    }

    /**
     * Disconnect WebSocket
     */
    disconnectWebSocket() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    /**
     * Register event listener
     * @param {string} event - Event name (job_status, batch_scan_progress, connected, disconnected, error)
     * @param {Function} callback - Callback function
     */
    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    /**
     * Remove event listener
     * @param {string} event - Event name
     * @param {Function} callback - Callback function to remove
     */
    off(event, callback) {
        if (!this.listeners[event]) return;

        const index = this.listeners[event].indexOf(callback);
        if (index > -1) {
            this.listeners[event].splice(index, 1);
        }
    }

    /**
     * Emit event to listeners
     * @private
     */
    _emit(event, data) {
        if (!this.listeners[event]) return;

        this.listeners[event].forEach(callback => {
            try {
                callback(data);
            } catch (error) {
                console.error(`Error in ${event} listener:`, error);
            }
        });
    }

    // ==================== Scanner API Methods ====================

    /**
     * Get list of available scanners
     * @returns {Promise<Array>} List of scanners
     */
    async listScanners() {
        const response = await fetch(`${this.apiBase}/scanners`);
        if (!response.ok) {
            throw new Error(`Failed to list scanners: ${response.statusText}`);
        }
        const data = await response.json();
        return data.scanners || [];
    }

    /**
     * Get specific scanner by ID
     * @param {string} scannerId - Scanner ID
     * @returns {Promise<Object>} Scanner details
     */
    async getScanner(scannerId) {
        const response = await fetch(`${this.apiBase}/scanners/${scannerId}`);
        if (!response.ok) {
            throw new Error(`Failed to get scanner: ${response.statusText}`);
        }
        return await response.json();
    }

    // ==================== Scan Job API Methods ====================

    /**
     * Create a new scan job
     * @param {string} scannerId - Scanner ID
     * @param {Object} parameters - Scan parameters
     * @param {number} parameters.resolution - Resolution in DPI (75, 100, 150, 200, 300, 600)
     * @param {string} parameters.color_mode - Color mode (Color, Grayscale, BlackAndWhite)
     * @param {string} parameters.format - Output format (JPEG, PNG, TIFF, BMP, PDF)
     * @param {string} parameters.page_size - Paper size (A4, A3, A5, Letter, Legal, B4, B5) (optional)
     * @param {number} parameters.jpeg_quality - JPEG quality 10-100 (default: 75)
     * @param {number} parameters.width - Width in mm (deprecated, use page_size)
     * @param {number} parameters.height - Height in mm (deprecated, use page_size)
     * @param {boolean} parameters.use_feeder - Use ADF (default: false)
     * @param {boolean} parameters.use_duplex - Use duplex (default: false)
     * @param {number} parameters.page_count - Number of pages (default: 1)
     * @returns {Promise<Object>} Created job
     */
    async createScan(scannerId, parameters) {
        const response = await fetch(`${this.apiBase}/scan`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                scanner_id: scannerId,
                parameters: {
                    resolution: parameters.resolution || 300,
                    color_mode: parameters.color_mode || 'Color',
                    format: parameters.format || 'JPEG',
                    page_size: parameters.page_size || '',
                    jpeg_quality: parameters.jpeg_quality || 75,
                    width: parameters.width || 210,
                    height: parameters.height || 297,
                    brightness: parameters.brightness || 0,
                    contrast: parameters.contrast || 0,
                    use_duplex: parameters.use_duplex || false,
                    use_feeder: parameters.use_feeder || false,
                    page_count: parameters.page_count || 1
                }
            })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to create scan');
        }

        return await response.json();
    }

    /**
     * Create a batch scan (multiple scans)
     * @param {string} scannerId - Scanner ID
     * @param {Object} parameters - Scan parameters (same as createScan)
     * @param {Object} batchSettings - Batch scan settings
     * @param {string} batchSettings.scan_type - Type: single, multiple_with_prompt, multiple_with_delay
     * @param {number} batchSettings.scan_count - Number of scans (for multiple types)
     * @param {number} batchSettings.scan_interval - Interval in seconds (for multiple_with_delay)
     * @returns {Promise<Object>} Batch scan result
     */
    async createBatchScan(scannerId, parameters, batchSettings) {
        const response = await fetch(`${this.apiBase}/scan/batch`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                scanner_id: scannerId,
                parameters: parameters,
                batch_settings: batchSettings
            })
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to create batch scan');
        }

        return await response.json();
    }

    /**
     * Get list of all scan jobs
     * @returns {Promise<Array>} List of jobs
     */
    async listJobs() {
        const response = await fetch(`${this.apiBase}/jobs`);
        if (!response.ok) {
            throw new Error(`Failed to list jobs: ${response.statusText}`);
        }
        const data = await response.json();
        return data.jobs || [];
    }

    /**
     * Get specific job by ID
     * @param {string} jobId - Job ID
     * @returns {Promise<Object>} Job details
     */
    async getJob(jobId) {
        const response = await fetch(`${this.apiBase}/jobs/${jobId}`);
        if (!response.ok) {
            throw new Error(`Failed to get job: ${response.statusText}`);
        }
        return await response.json();
    }

    /**
     * Cancel a running scan job
     * @param {string} jobId - Job ID
     * @returns {Promise<Object>} Cancellation result
     */
    async cancelJob(jobId) {
        const response = await fetch(`${this.apiBase}/jobs/${jobId}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to cancel job');
        }

        return await response.json();
    }

    // ==================== Utility Methods ====================

    /**
     * Get file URL for scanned result
     * @param {string} filePath - File path from scan result
     * @returns {string} Full URL to file
     */
    getFileUrl(filePath) {
        // Remove leading slash if present
        const cleanPath = filePath.replace(/^\//, '');
        return `${this.apiBase}/files/${cleanPath}`;
    }

    /**
     * Health check
     * @returns {Promise<Object>} Health status
     */
    async healthCheck() {
        const response = await fetch(`${this.apiBase}/health`);
        if (!response.ok) {
            throw new Error(`Health check failed: ${response.statusText}`);
        }
        return await response.json();
    }

    /**
     * Wait for job completion with polling
     * @param {string} jobId - Job ID
     * @param {number} pollInterval - Polling interval in ms (default: 1000)
     * @param {number} timeout - Timeout in ms (default: 300000 = 5 minutes)
     * @param {Function} onProgress - Progress callback (optional)
     * @returns {Promise<Object>} Completed job
     */
    async waitForJobCompletion(jobId, pollInterval = 1000, timeout = 300000, onProgress = null) {
        const startTime = Date.now();

        while (true) {
            // Check timeout
            if (Date.now() - startTime > timeout) {
                throw new Error('Job completion timeout');
            }

            const job = await this.getJob(jobId);

            // Call progress callback
            if (onProgress && job.status === 'processing') {
                onProgress(job.progress || 0);
            }

            // Check if completed
            if (job.status === 'completed') {
                return job;
            }

            if (job.status === 'failed') {
                throw new Error(`Scan failed: ${job.error}`);
            }

            if (job.status === 'cancelled') {
                throw new Error('Scan was cancelled');
            }

            // Wait before next poll
            await new Promise(resolve => setTimeout(resolve, pollInterval));
        }
    }
}

// Export for different module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ScannerClient;
}

if (typeof window !== 'undefined') {
    window.ScannerClient = ScannerClient;
}
