# Scanner Service JavaScript SDK

一个完整的JavaScript SDK，用于与扫描服务API交互。

## 📦 文件说明

- `scanner-sdk.js` - SDK主文件（可直接在浏览器或Node.js中使用）
- `SDK-README.md` - 完整的API文档和使用示例
- `example.html` - 交互式演示页面
- `package.json` - NPM包配置

## 🚀 快速开始

### 1. 在浏览器中使用

```html
<script src="scanner-sdk.js"></script>
<script>
    // 初始化客户端
    const client = new ScannerClient('http://localhost:8080');

    // 监听扫描任务更新
    client.on('job_status', (job) => {
        console.log('Job status:', job.status);
        console.log('Progress:', job.progress + '%');
    });

    // 获取扫描仪列表
    async function scan() {
        const scanners = await client.listScanners();
        console.log('Scanners:', scanners);

        // 开始扫描
        const job = await client.createScan(scanners[0].id, {
            resolution: 300,
            color_mode: 'Color',
            format: 'JPEG'
        });

        console.log('Scan started:', job.id);
    }

    scan();
</script>
```

### 2. 在Node.js中使用

```javascript
const ScannerClient = require('./scanner-sdk.js');

const client = new ScannerClient('http://localhost:8080');

async function main() {
    // 获取扫描仪
    const scanners = await client.listScanners();
    console.log('Available scanners:', scanners);

    // 创建扫描任务
    const job = await client.createScan(scanners[0].id, {
        resolution: 300,
        color_mode: 'Color',
        format: 'PDF',
        use_feeder: true
    });

    // 等待完成
    const result = await client.waitForJobCompletion(job.id);
    console.log('Scan completed:', result);

    // 获取文件URL
    result.results.forEach((r, i) => {
        const url = client.getFileUrl(r.file_path);
        console.log(`Page ${i + 1}: ${url}`);
    });
}

main();
```

### 3. 运行演示页面

1. 确保扫描服务正在运行（默认端口8080）
2. 在浏览器中打开 `example.html`
3. 修改页面中的 `SERVER_URL` 为你的服务器地址
4. 点击"刷新扫描仪列表"开始使用

## 📖 完整文档

查看 [SDK-README.md](./SDK-README.md) 获取：
- 完整的API文档
- 所有方法说明
- React/Vue.js集成示例
- 错误处理指南

## 🎯 主要功能

- ✅ 获取扫描仪列表
- ✅ 创建扫描任务
- ✅ 批量扫描支持
- ✅ WebSocket实时更新
- ✅ 进度监控
- ✅ 任务管理（列表、查询、取消）
- ✅ 文件URL生成
- ✅ 自动重连

## 📡 API方法

### 扫描仪相关
- `listScanners()` - 获取扫描仪列表
- `getScanner(id)` - 获取扫描仪详情

### 扫描任务相关
- `createScan(scannerId, parameters)` - 创建扫描
- `createBatchScan(scannerId, parameters, batchSettings)` - 批量扫描
- `listJobs()` - 获取任务列表
- `getJob(jobId)` - 获取任务详情
- `cancelJob(jobId)` - 取消任务
- `waitForJobCompletion(jobId, ...)` - 等待任务完成

### WebSocket事件
- `on('job_status', callback)` - 监听任务状态
- `on('batch_scan_progress', callback)` - 监听批量扫描进度
- `on('connected', callback)` - 连接成功
- `on('disconnected', callback)` - 连接断开
- `on('error', callback)` - 错误事件

### 工具方法
- `getFileUrl(filePath)` - 获取文件完整URL
- `healthCheck()` - 健康检查
- `connectWebSocket()` - 手动连接WebSocket
- `disconnectWebSocket()` - 断开WebSocket

## 💡 使用示例

### 简单扫描
```javascript
const client = new ScannerClient('http://localhost:8080');

const scanners = await client.listScanners();
const job = await client.createScan(scanners[0].id, {
    resolution: 300,
    format: 'JPEG'
});

// 等待完成并获取结果
const result = await client.waitForJobCompletion(job.id);
console.log('Scanned pages:', result.results.length);
```

### ADF批量扫描
```javascript
const job = await client.createScan(scannerId, {
    resolution: 300,
    format: 'PDF',
    use_feeder: true,  // 使用ADF
    page_count: 100    // 自动扫描所有页
});
```

### 监听实时进度
```javascript
client.on('job_status', (job) => {
    if (job.status === 'processing') {
        updateProgressBar(job.progress);
    } else if (job.status === 'completed') {
        showResults(job.results);
    }
});
```

## 🔧 配置选项

```javascript
const client = new ScannerClient('http://localhost:8080', {
    autoConnect: true,        // 自动连接WebSocket
    reconnectInterval: 3000   // 重连间隔（毫秒）
});
```

## 🌐 浏览器兼容性

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+
- 需要支持 WebSocket 和 Fetch API

## 📦 集成到你的项目

### 方式1: 直接复制
将 `scanner-sdk.js` 复制到你的项目中，然后引入：

```html
<script src="path/to/scanner-sdk.js"></script>
```

### 方式2: NPM包（如果发布）
```bash
npm install scanner-service-sdk
```

```javascript
import ScannerClient from 'scanner-service-sdk';
```

### 方式3: ES6模块
```javascript
import ScannerClient from './scanner-sdk.js';
```

## 🎨 React示例

```jsx
import { useState, useEffect } from 'react';
import ScannerClient from './scanner-sdk';

function App() {
    const [client] = useState(() => new ScannerClient('http://localhost:8080'));
    const [scanners, setScanners] = useState([]);

    useEffect(() => {
        client.listScanners().then(setScanners);

        client.on('job_status', (job) => {
            console.log('Job update:', job);
        });

        return () => client.disconnectWebSocket();
    }, [client]);

    const handleScan = async (scannerId) => {
        await client.createScan(scannerId, {
            resolution: 300,
            format: 'JPEG'
        });
    };

    return (
        <div>
            {scanners.map(s => (
                <button key={s.id} onClick={() => handleScan(s.id)}>
                    {s.name}
                </button>
            ))}
        </div>
    );
}
```

## 🔒 CORS跨域支持

**✅ 已内置CORS支持！**

从 v1.0.13+ 开始，服务器已配置CORS中间件，允许跨域访问。

**这意味着你可以：**
- 从不同端口访问（如：网站在3000端口，扫描服务在8080端口）
- 从不同域名访问（如：https://myapp.com 访问 http://192.168.1.100:8080）
- 无需额外配置

**示例：**
```javascript
// 你的网站: http://localhost:3000
// 扫描服务: http://localhost:8080
const client = new ScannerClient('http://localhost:8080');

// 可以正常工作，不会有CORS错误
const scanners = await client.listScanners();
```

**如果遇到CORS问题：**
- 查看详细指南：[CORS-GUIDE.md](./CORS-GUIDE.md)
- 确保使用 v1.0.13+ 版本
- 检查服务器是否正在运行
- 清除浏览器缓存

---

## 📄 许可证

MIT License

## 🤝 支持

如有问题，请查看：
- 完整文档: [SDK-README.md](./SDK-README.md)
- 演示页面: [example.html](./example.html)
