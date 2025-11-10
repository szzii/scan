/**
 * Scanner Service Client SDK
 * Supports both Node.js and Browser environments
 */

class ScannerClient {
    /**
     * Create a new ScannerClient instance
     * @param {string} baseURL - The base URL of the scanner service (e.g., 'http://localhost:8080')
     * @param {Object} options - Configuration options
     * @param {boolean} options.autoReconnect - Auto-reconnect WebSocket on disconnect (default: true)
     * @param {number} options.reconnectInterval - WebSocket reconnect interval in ms (default: 3000)
     */
    constructor(baseURL, options = {}) {
        this.baseURL = baseURL.replace(/\/$/, ''); // Remove trailing slash
        this.apiURL = `${this.baseURL}/api/v1`;
        this.wsURL = this.baseURL.replace('http', 'ws') + '/ws';

        this.options = {
            autoReconnect: options.autoReconnect !== false,
            reconnectInterval: options.reconnectInterval || 3000
        };

        this.ws = null;
        this.wsConnected = false;
        this.wsListeners = {
            job_status: [],
            scanner_status: [],
            error: [],
            connected: [],
            disconnected: []
        };
    }

    /**
     * Make an HTTP request to the API
     * @private
     */
    async _request(endpoint, options = {}) {
        const url = `${this.apiURL}${endpoint}`;

        try {
            const response = await fetch(url, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || `HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            throw new Error(`Request failed: ${error.message}`);
        }
    }

    /**
     * Get list of available scanners
     * @returns {Promise<Array>} Array of scanner objects
     */
    async listScanners() {
        const data = await this._request('/scanners');
        return data.scanners || [];
    }

    /**
     * Get details of a specific scanner
     * @param {string} scannerId - Scanner ID
     * @returns {Promise<Object>} Scanner object
     */
    async getScanner(scannerId) {
        return await this._request(`/scanners/${scannerId}`);
    }

    /**
     * Create a new scan job
     * @param {string} scannerId - Scanner ID
     * @param {Object} parameters - Scan parameters
     * @param {number} parameters.resolution - DPI (e.g., 300)
     * @param {string} parameters.color_mode - Color mode: 'Color', 'Grayscale', 'BlackAndWhite'
     * @param {string} parameters.format - Output format: 'PDF', 'JPEG', 'PNG', 'TIFF'
     * @param {number} parameters.width - Width in mm (default: 210 for A4)
     * @param {number} parameters.height - Height in mm (default: 297 for A4)
     * @param {number} parameters.page_count - Number of pages (0 for single page)
     * @param {boolean} parameters.use_duplex - Use duplex scanning
     * @param {boolean} parameters.use_feeder - Use document feeder
     * @returns {Promise<Object>} Created scan job
     */
    async createScan(scannerId, parameters) {
        const defaultParams = {
            resolution: 300,
            color_mode: 'Color',
            format: 'PDF',
            width: 210,
            height: 297,
            brightness: 0,
            contrast: 0,
            use_duplex: false,
            use_feeder: false,
            page_count: 1
        };

        return await this._request('/scan', {
            method: 'POST',
            body: JSON.stringify({
                scanner_id: scannerId,
                parameters: { ...defaultParams, ...parameters }
            })
        });
    }

    /**
     * Create a batch scan job
     * @param {string} scannerId - Scanner ID
     * @param {Object} parameters - Scan parameters
     * @param {number} batchCount - Number of scans to perform
     * @returns {Promise<Object>} Response with job IDs
     */
    async createBatchScan(scannerId, parameters, batchCount) {
        const defaultParams = {
            resolution: 300,
            color_mode: 'Color',
            format: 'PDF',
            width: 210,
            height: 297,
            brightness: 0,
            contrast: 0,
            use_duplex: false,
            use_feeder: false,
            page_count: 1
        };

        return await this._request('/scan/batch', {
            method: 'POST',
            body: JSON.stringify({
                scanner_id: scannerId,
                parameters: { ...defaultParams, ...parameters },
                batch_count: batchCount
            })
        });
    }

    /**
     * Get list of all scan jobs
     * @returns {Promise<Array>} Array of job objects
     */
    async listJobs() {
        const data = await this._request('/jobs');
        return data.jobs || [];
    }

    /**
     * Get details of a specific job
     * @param {string} jobId - Job ID
     * @returns {Promise<Object>} Job object
     */
    async getJob(jobId) {
        return await this._request(`/jobs/${jobId}`);
    }

    /**
     * Cancel a running scan job
     * @param {string} jobId - Job ID
     * @returns {Promise<Object>} Response message
     */
    async cancelJob(jobId) {
        return await this._request(`/jobs/${jobId}`, {
            method: 'DELETE'
        });
    }

    /**
     * Check service health
     * @returns {Promise<Object>} Health status
     */
    async healthCheck() {
        return await this._request('/health');
    }

    /**
     * Connect to WebSocket for real-time updates
     * @returns {Promise<void>}
     */
    connectWebSocket() {
        return new Promise((resolve, reject) => {
            if (this.ws && this.wsConnected) {
                resolve();
                return;
            }

            try {
                this.ws = new WebSocket(this.wsURL);

                this.ws.onopen = () => {
                    this.wsConnected = true;
                    this._emitEvent('connected');
                    resolve();
                };

                this.ws.onclose = () => {
                    this.wsConnected = false;
                    this._emitEvent('disconnected');

                    if (this.options.autoReconnect) {
                        setTimeout(() => {
                            this.connectWebSocket();
                        }, this.options.reconnectInterval);
                    }
                };

                this.ws.onerror = (error) => {
                    this._emitEvent('error', error);
                    reject(error);
                };

                this.ws.onmessage = (event) => {
                    try {
                        const message = JSON.parse(event.data);
                        this._handleWebSocketMessage(message);
                    } catch (error) {
                        console.error('Failed to parse WebSocket message:', error);
                    }
                };
            } catch (error) {
                reject(error);
            }
        });
    }

    /**
     * Disconnect WebSocket
     */
    disconnectWebSocket() {
        if (this.ws) {
            this.options.autoReconnect = false;
            this.ws.close();
            this.ws = null;
            this.wsConnected = false;
        }
    }

    /**
     * Handle incoming WebSocket message
     * @private
     */
    _handleWebSocketMessage(message) {
        const { type, payload } = message;

        if (this.wsListeners[type]) {
            this.wsListeners[type].forEach(callback => {
                try {
                    callback(payload);
                } catch (error) {
                    console.error(`Error in WebSocket listener for ${type}:`, error);
                }
            });
        }
    }

    /**
     * Emit an event to registered listeners
     * @private
     */
    _emitEvent(eventType, data = null) {
        if (this.wsListeners[eventType]) {
            this.wsListeners[eventType].forEach(callback => {
                try {
                    callback(data);
                } catch (error) {
                    console.error(`Error in event listener for ${eventType}:`, error);
                }
            });
        }
    }

    /**
     * Register a WebSocket event listener
     * @param {string} eventType - Event type: 'job_status', 'scanner_status', 'error', 'connected', 'disconnected'
     * @param {Function} callback - Callback function
     */
    on(eventType, callback) {
        if (!this.wsListeners[eventType]) {
            this.wsListeners[eventType] = [];
        }
        this.wsListeners[eventType].push(callback);
    }

    /**
     * Remove a WebSocket event listener
     * @param {string} eventType - Event type
     * @param {Function} callback - Callback function to remove
     */
    off(eventType, callback) {
        if (this.wsListeners[eventType]) {
            this.wsListeners[eventType] = this.wsListeners[eventType].filter(cb => cb !== callback);
        }
    }

    /**
     * Remove all listeners for an event type
     * @param {string} eventType - Event type
     */
    removeAllListeners(eventType) {
        if (eventType) {
            this.wsListeners[eventType] = [];
        } else {
            Object.keys(this.wsListeners).forEach(key => {
                this.wsListeners[key] = [];
            });
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
