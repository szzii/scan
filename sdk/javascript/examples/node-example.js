/**
 * Node.js Example for Scanner Service Client SDK
 *
 * Usage:
 *   node node-example.js
 */

// For Node.js, you need to install node-fetch and ws:
// npm install node-fetch ws

// Polyfill fetch for Node.js < 18
if (typeof fetch === 'undefined') {
    global.fetch = require('node-fetch');
}

// Polyfill WebSocket for Node.js
if (typeof WebSocket === 'undefined') {
    global.WebSocket = require('ws');
}

const ScannerClient = require('../src/scanner-client');

// Configuration
const SERVICE_URL = 'http://localhost:8080';

async function main() {
    console.log('Scanner Service Client - Node.js Example\n');

    // Create client instance
    const client = new ScannerClient(SERVICE_URL);

    try {
        // 1. Check service health
        console.log('1. Checking service health...');
        const health = await client.healthCheck();
        console.log('Service status:', health.status);
        console.log('');

        // 2. List available scanners
        console.log('2. Listing available scanners...');
        const scanners = await client.listScanners();
        console.log(`Found ${scanners.length} scanner(s):`);
        scanners.forEach(scanner => {
            console.log(`  - ${scanner.name} (${scanner.id})`);
            console.log(`    Model: ${scanner.manufacturer} ${scanner.model}`);
            console.log(`    Status: ${scanner.status}`);
        });
        console.log('');

        if (scanners.length === 0) {
            console.log('No scanners available. Exiting.');
            return;
        }

        // 3. Connect WebSocket for real-time updates
        console.log('3. Connecting to WebSocket...');
        await client.connectWebSocket();
        console.log('WebSocket connected!');
        console.log('');

        // Register event listeners
        client.on('job_status', (job) => {
            console.log(`[WebSocket] Job ${job.id.substring(0, 8)}... status: ${job.status} (${job.progress}%)`);
        });

        client.on('scanner_status', (scanner) => {
            console.log(`[WebSocket] Scanner ${scanner.id} status changed: ${scanner.status}`);
        });

        client.on('disconnected', () => {
            console.log('[WebSocket] Disconnected');
        });

        // 4. Create a scan job
        const scanner = scanners[0];
        console.log(`4. Creating scan job with scanner: ${scanner.name}...`);

        const scanParams = {
            resolution: 300,
            color_mode: 'Color',
            format: 'PDF',
            page_count: 1
        };

        const job = await client.createScan(scanner.id, scanParams);
        console.log('Scan job created:');
        console.log(`  Job ID: ${job.id}`);
        console.log(`  Status: ${job.status}`);
        console.log(`  Parameters: ${job.parameters.resolution} DPI, ${job.parameters.color_mode}, ${job.parameters.format}`);
        console.log('');

        // 5. Monitor job progress
        console.log('5. Monitoring job progress...');
        console.log('(Real-time updates will appear via WebSocket)');
        console.log('');

        // Wait for job completion
        await waitForJobCompletion(client, job.id);

        // 6. Get final job details
        console.log('6. Getting final job details...');
        const completedJob = await client.getJob(job.id);
        console.log(`Job ${completedJob.id} completed!`);
        console.log(`  Status: ${completedJob.status}`);
        if (completedJob.results && completedJob.results.length > 0) {
            console.log(`  Results:`);
            completedJob.results.forEach(result => {
                console.log(`    - Page ${result.page_number}: ${result.file_path} (${formatFileSize(result.file_size)})`);
            });
        }
        if (completedJob.error) {
            console.log(`  Error: ${completedJob.error}`);
        }
        console.log('');

        // 7. Example: Create batch scan
        console.log('7. Example: Creating batch scan (3 scans)...');
        const batchResult = await client.createBatchScan(scanner.id, scanParams, 3);
        console.log('Batch scan created:');
        console.log(`  Job IDs: ${batchResult.job_ids.join(', ')}`);
        console.log('');

        // Wait a bit to see WebSocket updates
        console.log('Waiting for batch jobs to complete...');
        await new Promise(resolve => setTimeout(resolve, 10000));

        // 8. List all jobs
        console.log('8. Listing all jobs...');
        const allJobs = await client.listJobs();
        console.log(`Total jobs: ${allJobs.length}`);
        allJobs.forEach(j => {
            console.log(`  - ${j.id.substring(0, 8)}...: ${j.status} (${j.progress}%)`);
        });
        console.log('');

    } catch (error) {
        console.error('Error:', error.message);
    } finally {
        // Cleanup
        console.log('Disconnecting WebSocket...');
        client.disconnectWebSocket();
        console.log('Done!');
    }
}

/**
 * Wait for a job to complete
 */
async function waitForJobCompletion(client, jobId, timeout = 60000) {
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
        const job = await client.getJob(jobId);

        if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
            return job;
        }

        await new Promise(resolve => setTimeout(resolve, 1000));
    }

    throw new Error('Job timeout');
}

/**
 * Format file size
 */
function formatFileSize(bytes) {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

// Run the example
if (require.main === module) {
    main().catch(console.error);
}
