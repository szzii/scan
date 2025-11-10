# ADF 高速扫描快速测试指南

## 测试前准备

### 1. 硬件要求
- ✅ 支持 ADF 的扫描仪（自动送纸器）
- ✅ 测试用纸张（建议 10-20 张）
- ✅ Windows 10/11 系统

### 2. 软件准备
```bash
# 解压 Windows 版本
unzip scanserver-windows-amd64-v1.0.0.zip
cd scanserver-windows-amd64-v1.0.0

# 运行服务
scanserver.exe
```

服务会启动在 `http://localhost:8080`

## 测试步骤

### 测试 1：基本 ADF 扫描（10 页）

**目的：** 验证 ADF 批量扫描功能

1. **打开浏览器**
   ```
   http://localhost:8080
   ```

2. **配置扫描参数**
   - Resolution: `300 DPI`
   - Color Mode: `Grayscale`
   - Format: `JPEG`
   - Pages: `10`
   - ✅ 勾选 "Use Auto Document Feeder (ADF)"

3. **准备扫描仪**
   - 在送纸器中放入 10 张纸
   - 确保纸张整齐、无钉书钉

4. **开始扫描**
   - 点击 "Start Scan" 按钮
   - 观察进度条
   - 扫描完成后查看图片预览

**预期结果：**
- ✅ 扫描仪连续扫描 10 张纸
- ✅ 无明显停顿或延迟
- ✅ 生成 10 个文件：`scan_YYYYMMDD_HHMMSS_page_1.jpg` ~ `page_10.jpg`
- ✅ 每个文件都能在浏览器中预览

### 测试 2：高速扫描（50 页）

**目的：** 测试高速连续扫描性能

1. **配置参数**
   - Resolution: `200 DPI` (更快)
   - Color Mode: `Grayscale`
   - Format: `JPEG`
   - Pages: `50`
   - ✅ 勾选 "Use Auto Document Feeder (ADF)"

2. **放入 50 张纸**

3. **开始扫描并计时**
   - 记录开始时间
   - 点击 "Start Scan"
   - 记录结束时间

**预期结果：**
- ✅ 扫描速度接近扫描仪硬件规格（例如 40 ppm = 75 秒）
- ✅ 没有每页之间的停顿
- ✅ 生成 50 个文件

**参考时间：**
| 扫描仪速度 | 预期时间（50 页）|
|-----------|----------------|
| 40 ppm    | ~75 秒         |
| 60 ppm    | ~50 秒         |
| 80 ppm    | ~38 秒         |

### 测试 3：双面扫描

**目的：** 测试双面扫描功能（如果扫描仪支持）

1. **配置参数**
   - Resolution: `300 DPI`
   - Color Mode: `Grayscale`
   - Format: `JPEG`
   - Pages: `10` (10 张纸 = 20 面)
   - ✅ 勾选 "Use Auto Document Feeder (ADF)"
   - ✅ 勾选 "Duplex Scanning"

2. **放入 10 张双面纸**

3. **开始扫描**

**预期结果：**
- ✅ 扫描仪扫描纸张的正反两面
- ✅ 生成 20 个文件（或 10 个双面文件，取决于驱动）

### 测试 4：API 调用测试

**目的：** 验证 API 接口工作正常

```powershell
# PowerShell 命令
$body = @{
    scanner_id = "your-scanner-id"  # 先通过 /api/v1/scanners 获取
    parameters = @{
        resolution = 300
        color_mode = "Grayscale"
        format = "JPEG"
        use_feeder = $true
        page_count = 5
    }
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/v1/scan" `
    -Method POST `
    -ContentType "application/json" `
    -Body $body
```

**预期结果：**
```json
{
  "id": "job-xxx",
  "status": "completed",
  "results": [
    {"page_number": 1, "file_path": "scans/scan_xxx_page_1.jpg"},
    {"page_number": 2, "file_path": "scans/scan_xxx_page_2.jpg"},
    ...
  ]
}
```

## 性能测试

### 测试场景：100 页高速扫描

**配置：**
```json
{
  "resolution": 200,
  "color_mode": "Grayscale",
  "format": "JPEG",
  "use_feeder": true,
  "page_count": 100
}
```

**测量指标：**
1. **总耗时** - 从开始到完成的时间
2. **平均每页时间** - 总耗时 / 100
3. **实际 ppm** - 60 / 平均每页时间（秒）

**记录结果：**
```
扫描仪型号: _______________
扫描页数: 100
分辨率: 200 DPI
颜色模式: Grayscale

总耗时: _____ 秒
平均每页: _____ 秒
实际 ppm: _____ 页/分钟

理论 ppm (扫描仪规格): _____ 页/分钟
效率: _____% (实际/理论)
```

**目标：**
- ✅ 效率 > 90% - 优秀
- ✅ 效率 > 80% - 良好
- ⚠️ 效率 < 80% - 需要调查

## 对比测试：有延迟 vs 无延迟

如果您想对比优化前后的性能：

### 模拟延迟版本
临时在代码中加入延迟（仅用于测试对比）：
```go
// 在 driver_windows.go 的 for 循环结尾添加
time.Sleep(100 * time.Millisecond)
```

### 测试对比
| 版本 | 10 页 | 50 页 | 100 页 |
|------|-------|-------|--------|
| 有延迟 (100ms) | ___s | ___s | ___s |
| 无延迟 | ___s | ___s | ___s |
| 性能提升 | ___% | ___% | ___% |

## 故障排查

### 问题 1：扫描仍然很慢

**检查清单：**
```bash
# 1. 查看扫描仪规格
curl http://localhost:8080/api/v1/scanners | jq

# 2. 检查系统日志
# 在 scanserver 控制台查看是否有错误信息

# 3. 测试不同分辨率
# 尝试 150 DPI, 200 DPI, 300 DPI

# 4. 测试不同颜色模式
# BlackAndWhite > Grayscale > Color
```

### 问题 2：某些页面缺失

**可能原因：**
- 送纸器中多张纸粘连
- 扫描仪传感器脏污
- 纸张质量问题

**解决方案：**
```bash
# 1. 清洁扫描仪
# 参考扫描仪手册清洁滚轮和传感器

# 2. 准备纸张
# 将纸张扇开，避免粘连

# 3. 减少每批次的页数
# 从 100 页改为 50 页一批
```

### 问题 3：文件太大

**优化建议：**
```json
{
  "resolution": 150,           // 降低分辨率
  "color_mode": "Grayscale",   // 灰度而非彩色
  "format": "JPEG"             // JPEG 压缩
}
```

**文件大小参考：**
| 分辨率 | 颜色模式 | 大小/页 |
|--------|---------|--------|
| 150 DPI | B&W | ~50 KB |
| 200 DPI | Grayscale | ~150 KB |
| 300 DPI | Grayscale | ~200 KB |
| 300 DPI | Color | ~500 KB |
| 600 DPI | Color | ~2 MB |

## WebSocket 实时监控

使用浏览器控制台监控扫描进度：

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
    console.log('WebSocket connected');
};

ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    console.log('Message:', msg);

    if (msg.type === 'job_status') {
        console.log(`Job ${msg.payload.id}: ${msg.payload.progress}%`);
        console.log(`Pages scanned: ${msg.payload.results.length}`);
    }
};

ws.onerror = (error) => {
    console.error('WebSocket error:', error);
};
```

## 批量自动化测试

PowerShell 脚本自动测试多种配置：

```powershell
# test-adf-speeds.ps1

$configurations = @(
    @{name="Fast B&W 150dpi"; resolution=150; color="BlackAndWhite"},
    @{name="Standard Gray 200dpi"; resolution=200; color="Grayscale"},
    @{name="Quality Gray 300dpi"; resolution=300; color="Grayscale"},
    @{name="High Quality Color 300dpi"; resolution=300; color="Color"}
)

$scannerId = "your-scanner-id"  # 替换为实际扫描仪 ID
$pageCount = 10

foreach ($config in $configurations) {
    Write-Host "`n========================================" -ForegroundColor Green
    Write-Host "Testing: $($config.name)" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green

    Write-Host "Please load $pageCount pages and press Enter..."
    Read-Host

    $body = @{
        scanner_id = $scannerId
        parameters = @{
            resolution = $config.resolution
            color_mode = $config.color
            format = "JPEG"
            use_feeder = $true
            page_count = $pageCount
        }
    } | ConvertTo-Json

    $startTime = Get-Date

    try {
        $result = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/scan" `
            -Method POST `
            -ContentType "application/json" `
            -Body $body

        $endTime = Get-Date
        $duration = ($endTime - $startTime).TotalSeconds

        Write-Host "Status: SUCCESS" -ForegroundColor Green
        Write-Host "Duration: $duration seconds"
        Write-Host "Pages: $($result.results.Length)"
        Write-Host "Avg per page: $([math]::Round($duration / $pageCount, 2)) seconds"
    }
    catch {
        Write-Host "Status: FAILED" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)"
    }
}

Write-Host "`nAll tests completed!" -ForegroundColor Green
```

## 成功标准

扫描测试通过的标准：

- ✅ **功能性**
  - ADF 连续扫描工作正常
  - 所有页面都被正确扫描
  - 文件正确保存到 scans/ 目录
  - 浏览器预览功能正常

- ✅ **性能**
  - 扫描速度接近扫描仪规格（效率 > 80%）
  - 页面之间无明显停顿
  - 进度条实时更新

- ✅ **稳定性**
  - 连续扫描 100 页无错误
  - 扫描仪不卡纸
  - 服务不崩溃

- ✅ **易用性**
  - Web 界面简单直观
  - API 调用简单明了
  - 错误信息清晰有用

## 反馈

如果测试中发现问题，请记录：

1. **扫描仪型号**：_______________
2. **Windows 版本**：_______________
3. **问题描述**：_______________
4. **复现步骤**：_______________
5. **期望行为**：_______________
6. **实际行为**：_______________
7. **日志输出**：_______________

祝测试顺利！🚀
