# Scanner Service JavaScript SDK

一个完整的JavaScript SDK，用于与扫描服务API交互。

## 特性

- ✅ 完整的API封装
- ✅ WebSocket实时更新
- ✅ 事件监听系统
- ✅ Promise-based异步调用
- ✅ 自动重连机制
- ✅ 浏览器和Node.js兼容
- ✅ 无依赖，纯JavaScript

## 安装

### 浏览器中使用

```html
<script src="scanner-sdk.js"></script>
<script>
    const client = new ScannerClient('http://localhost:8080');
</script>
```

### Node.js中使用

```javascript
const ScannerClient = require('./scanner-sdk.js');
const client = new ScannerClient('http://localhost:8080');
```

## 快速开始

### 1. 初始化客户端

```javascript
// 创建客户端实例
const client = new ScannerClient('http://localhost:8080', {
    autoConnect: true,        // 自动连接WebSocket (默认: true)
    reconnectInterval: 3000   // 重连间隔 (默认: 3000ms)
});
```

### 2. 监听事件

```javascript
// 监听扫描任务状态更新
client.on('job_status', (job) => {
    console.log('Job update:', job);
    console.log(`Status: ${job.status}, Progress: ${job.progress}%`);
});

// 监听批量扫描进度
client.on('batch_scan_progress', (progress) => {
    console.log('Batch progress:', progress);
});

// 监听连接状态
client.on('connected', () => {
    console.log('WebSocket connected');
});

client.on('disconnected', () => {
    console.log('WebSocket disconnected');
});

client.on('error', (error) => {
    console.error('Error:', error);
});
```

### 3. 获取扫描仪列表

```javascript
async function loadScanners() {
    try {
        const scanners = await client.listScanners();
        console.log('Available scanners:', scanners);

        scanners.forEach(scanner => {
            console.log(`- ${scanner.name} (${scanner.id})`);
        });
    } catch (error) {
        console.error('Failed to load scanners:', error);
    }
}
```

### 4. 创建扫描任务

```javascript
async function startScan() {
    try {
        const job = await client.createScan('wia:scanner-id', {
            resolution: 300,
            color_mode: 'Color',
            format: 'JPEG',
            use_feeder: false
        });

        console.log('Scan started:', job.id);

        // 等待扫描完成
        const completedJob = await client.waitForJobCompletion(
            job.id,
            1000,  // 每1秒检查一次
            300000, // 5分钟超时
            (progress) => {
                console.log(`Progress: ${progress}%`);
            }
        );

        console.log('Scan completed!');
        console.log('Results:', completedJob.results);

        // 获取扫描文件URL
        completedJob.results.forEach((result, index) => {
            const fileUrl = client.getFileUrl(result.file_path);
            console.log(`Page ${index + 1}: ${fileUrl}`);
        });

    } catch (error) {
        console.error('Scan failed:', error);
    }
}
```

## API文档

### 构造函数

#### `new ScannerClient(baseUrl, options)`

创建扫描客户端实例

**参数:**
- `baseUrl` (string): 扫描服务地址，例如 `http://localhost:8080`
- `options` (Object, 可选):
  - `autoConnect` (boolean): 自动连接WebSocket，默认 `true`
  - `reconnectInterval` (number): WebSocket重连间隔（毫秒），默认 `3000`

**示例:**
```javascript
const client = new ScannerClient('http://192.168.1.100:8080', {
    autoConnect: true,
    reconnectInterval: 5000
});
```

---

### WebSocket方法

#### `connectWebSocket()`

手动连接WebSocket（如果autoConnect为false）

#### `disconnectWebSocket()`

断开WebSocket连接

#### `on(event, callback)`

注册事件监听器

**事件类型:**
- `job_status`: 扫描任务状态更新
- `batch_scan_progress`: 批量扫描进度
- `connected`: WebSocket已连接
- `disconnected`: WebSocket已断开
- `error`: 错误事件

**示例:**
```javascript
client.on('job_status', (job) => {
    console.log(`Job ${job.id}: ${job.status}`);
});
```

#### `off(event, callback)`

移除事件监听器

---

### 扫描仪API

#### `listScanners()`

获取所有可用扫描仪

**返回:** `Promise<Array>`

**示例:**
```javascript
const scanners = await client.listScanners();
```

#### `getScanner(scannerId)`

获取指定扫描仪详情

**参数:**
- `scannerId` (string): 扫描仪ID

**返回:** `Promise<Object>`

---

### 扫描任务API

#### `createScan(scannerId, parameters)`

创建单次扫描任务

**参数:**
- `scannerId` (string): 扫描仪ID
- `parameters` (Object):
  - `resolution` (number): 分辨率 (75, 100, 150, 200, 300, 600)
  - `color_mode` (string): 颜色模式 ('Color', 'Grayscale', 'BlackAndWhite')
  - `format` (string): 输出格式 ('JPEG', 'PNG', 'TIFF', 'BMP', 'PDF')
  - `width` (number): 宽度（毫米），默认 210
  - `height` (number): 高度（毫米），默认 297
  - `use_feeder` (boolean): 使用ADF，默认 false
  - `use_duplex` (boolean): 使用双面，默认 false
  - `page_count` (number): 页数，默认 1

**返回:** `Promise<Object>` - 创建的任务对象

**示例:**
```javascript
const job = await client.createScan('wia:12345', {
    resolution: 300,
    color_mode: 'Color',
    format: 'PDF',
    use_feeder: true
});
```

#### `createBatchScan(scannerId, parameters, batchSettings)`

创建批量扫描任务

**参数:**
- `scannerId` (string): 扫描仪ID
- `parameters` (Object): 扫描参数（同createScan）
- `batchSettings` (Object):
  - `scan_type` (string): 扫描类型
    - `'single'`: 单次扫描
    - `'multiple_with_prompt'`: 多次扫描（每次提示）
    - `'multiple_with_delay'`: 多次扫描（固定延迟）
  - `scan_count` (number): 扫描次数
  - `scan_interval` (number): 扫描间隔（秒，用于multiple_with_delay）

**示例:**
```javascript
const result = await client.createBatchScan(
    'wia:12345',
    { resolution: 300, format: 'JPEG' },
    {
        scan_type: 'multiple_with_delay',
        scan_count: 5,
        scan_interval: 3
    }
);
```

#### `listJobs()`

获取所有扫描任务

**返回:** `Promise<Array>`

#### `getJob(jobId)`

获取指定任务详情

**参数:**
- `jobId` (string): 任务ID

**返回:** `Promise<Object>`

#### `cancelJob(jobId)`

取消正在运行的任务

**参数:**
- `jobId` (string): 任务ID

**返回:** `Promise<Object>`

---

### 工具方法

#### `getFileUrl(filePath)`

获取扫描文件的完整URL

**参数:**
- `filePath` (string): 扫描结果中的文件路径

**返回:** `string` - 完整的文件URL

**示例:**
```javascript
const url = client.getFileUrl('scans/scan_123.jpg');
// 返回: http://localhost:8080/api/v1/files/scans/scan_123.jpg
```

#### `healthCheck()`

健康检查

**返回:** `Promise<Object>`

#### `waitForJobCompletion(jobId, pollInterval, timeout, onProgress)`

等待任务完成（轮询方式）

**参数:**
- `jobId` (string): 任务ID
- `pollInterval` (number): 轮询间隔（毫秒），默认 1000
- `timeout` (number): 超时时间（毫秒），默认 300000 (5分钟)
- `onProgress` (Function): 进度回调，可选

**返回:** `Promise<Object>` - 完成的任务对象

**示例:**
```javascript
const job = await client.waitForJobCompletion(
    'job-123',
    1000,
    300000,
    (progress) => console.log(`${progress}%`)
);
```

---

## 完整示例

### 示例1: 简单扫描

```javascript
const client = new ScannerClient('http://localhost:8080');

// 监听任务更新
client.on('job_status', (job) => {
    console.log(`${job.status}: ${job.progress}%`);

    if (job.status === 'completed') {
        console.log('扫描完成！');
        job.results.forEach((result, i) => {
            const url = client.getFileUrl(result.file_path);
            console.log(`第${i+1}页: ${url}`);
        });
    }
});

// 开始扫描
async function scan() {
    const scanners = await client.listScanners();
    const scannerId = scanners[0].id;

    await client.createScan(scannerId, {
        resolution: 300,
        color_mode: 'Color',
        format: 'JPEG'
    });
}

scan();
```

### 示例2: ADF批量扫描

```javascript
async function batchScan() {
    const client = new ScannerClient('http://localhost:8080');

    // 监听进度
    client.on('job_status', (job) => {
        if (job.status === 'processing') {
            updateProgressBar(job.progress);
        }
    });

    const scanners = await client.listScanners();
    const scannerId = scanners[0].id;

    // 使用ADF扫描多页
    const job = await client.createScan(scannerId, {
        resolution: 300,
        color_mode: 'Color',
        format: 'PDF',
        use_feeder: true,
        page_count: 100  // ADF会自动扫描所有页
    });

    // 等待完成
    const result = await client.waitForJobCompletion(job.id);

    console.log(`扫描了 ${result.results.length} 页`);
    return result;
}
```

### 示例3: React集成

```jsx
import React, { useEffect, useState } from 'react';
import ScannerClient from './scanner-sdk';

function ScannerComponent() {
    const [client] = useState(() => new ScannerClient('http://localhost:8080'));
    const [scanners, setScanners] = useState([]);
    const [progress, setProgress] = useState(0);

    useEffect(() => {
        // 加载扫描仪列表
        client.listScanners().then(setScanners);

        // 监听进度更新
        const handleProgress = (job) => {
            setProgress(job.progress);
        };

        client.on('job_status', handleProgress);

        return () => {
            client.off('job_status', handleProgress);
            client.disconnectWebSocket();
        };
    }, [client]);

    const handleScan = async (scannerId) => {
        const job = await client.createScan(scannerId, {
            resolution: 300,
            format: 'JPEG'
        });

        const result = await client.waitForJobCompletion(job.id);
        alert(`扫描完成！共 ${result.results.length} 页`);
    };

    return (
        <div>
            <h2>扫描仪列表</h2>
            {scanners.map(scanner => (
                <button key={scanner.id} onClick={() => handleScan(scanner.id)}>
                    {scanner.name}
                </button>
            ))}
            {progress > 0 && <progress value={progress} max="100" />}
        </div>
    );
}
```

### 示例4: Vue.js集成

```vue
<template>
    <div>
        <select v-model="selectedScanner">
            <option v-for="scanner in scanners" :key="scanner.id" :value="scanner.id">
                {{ scanner.name }}
            </option>
        </select>

        <button @click="startScan" :disabled="!selectedScanner">开始扫描</button>

        <div v-if="scanning">
            <progress :value="progress" max="100"></progress>
            <span>{{ progress }}%</span>
        </div>

        <div v-if="results.length > 0">
            <h3>扫描结果</h3>
            <img v-for="(result, i) in results"
                 :key="i"
                 :src="getFileUrl(result.file_path)"
                 style="max-width: 200px; margin: 10px" />
        </div>
    </div>
</template>

<script>
import ScannerClient from './scanner-sdk';

export default {
    data() {
        return {
            client: null,
            scanners: [],
            selectedScanner: '',
            scanning: false,
            progress: 0,
            results: []
        };
    },

    mounted() {
        this.client = new ScannerClient('http://localhost:8080');

        this.client.on('job_status', (job) => {
            if (job.status === 'processing') {
                this.progress = job.progress;
            } else if (job.status === 'completed') {
                this.scanning = false;
                this.results = job.results;
            }
        });

        this.loadScanners();
    },

    beforeUnmount() {
        this.client.disconnectWebSocket();
    },

    methods: {
        async loadScanners() {
            this.scanners = await this.client.listScanners();
        },

        async startScan() {
            this.scanning = true;
            this.progress = 0;
            this.results = [];

            await this.client.createScan(this.selectedScanner, {
                resolution: 300,
                color_mode: 'Color',
                format: 'JPEG'
            });
        },

        getFileUrl(path) {
            return this.client.getFileUrl(path);
        }
    }
};
</script>
```

## 错误处理

```javascript
try {
    const job = await client.createScan(scannerId, parameters);
} catch (error) {
    if (error.message.includes('scanner is busy')) {
        alert('扫描仪正在使用中');
    } else if (error.message.includes('not found')) {
        alert('扫描仪未找到');
    } else {
        alert('扫描失败: ' + error.message);
    }
}
```

## 浏览器兼容性

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- 需要支持 WebSocket 和 Fetch API

## 许可证

MIT License

## 支持

如有问题，请提交 Issue。
