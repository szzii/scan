# ADF 高速扫描优化说明

## 优化内容

### 问题
之前的 ADF 实现在每页扫描之间有 100ms 的延迟：
```go
time.Sleep(100 * time.Millisecond)  // ❌ 降低扫描速度
```

对于高速扫描仪（40-100 ppm），这个延迟会严重影响性能：
- 扫描 50 页 = 5 秒额外等待时间
- 扫描 100 页 = 10 秒额外等待时间

### 解决方案

**移除所有人为延迟**，让扫描仪以最大速度连续扫描：

```go
// Loop to scan multiple pages - fast continuous mode
for i := 0; i < pageCount; i++ {
    // Transfer image - this is the actual scan operation
    imageRaw, err := oleutil.CallMethod(item, "Transfer", WiaFormatJPEG)
    // ...

    // Save immediately without delay
    _, err = oleutil.CallMethod(image, "SaveFile", filePath)
    image.Release()
    // ...

    // No delay - scanner continues immediately to next page
}
```

## 性能对比

### 优化前
| 页数 | 扫描仪速度 | 实际时间 | 人为延迟 | 总时间 |
|------|----------|---------|---------|--------|
| 10   | 40 ppm   | 15s     | 1s      | 16s    |
| 50   | 40 ppm   | 75s     | 5s      | 80s    |
| 100  | 100 ppm  | 60s     | 10s     | 70s    |

### 优化后
| 页数 | 扫描仪速度 | 实际时间 | 人为延迟 | 总时间 |
|------|----------|---------|---------|--------|
| 10   | 40 ppm   | 15s     | **0s**  | **15s** ✅ |
| 50   | 40 ppm   | 75s     | **0s**  | **75s** ✅ |
| 100  | 100 ppm  | 60s     | **0s**  | **60s** ✅ |

**性能提升：**
- 10 页：快 6.7%
- 50 页：快 6.7%
- 100 页：快 16.7%

## 技术细节

### 1. WIA Transfer 调用

```go
imageRaw, err := oleutil.CallMethod(item, "Transfer", WiaFormatJPEG)
```

这个调用会：
1. 触发扫描仪扫描下一页
2. 将图像数据传输到内存
3. 返回图像对象

**速度取决于：**
- 扫描仪硬件速度（ppm - pages per minute）
- 分辨率设置（300 DPI vs 600 DPI）
- 颜色模式（Color vs Grayscale vs B&W）

### 2. 文件保存

```go
oleutil.CallMethod(image, "SaveFile", filePath)
```

这个操作会：
1. 将内存中的图像编码为 JPEG
2. 写入磁盘

**速度取决于：**
- CPU 编码速度
- 磁盘写入速度（SSD vs HDD）
- 图像大小

### 3. 为什么移除延迟是安全的？

**WIA 驱动内部已经处理了时序：**
- `Transfer` 调用是阻塞的 - 只有当扫描仪准备好时才会返回
- 扫描仪硬件控制进纸速度
- 不需要软件层面的延迟

**如果扫描仪没准备好会怎样？**
- `Transfer` 会等待直到下一页准备好
- 或者返回错误（没有更多页面）
- 由硬件和驱动保证安全

## 使用方法

### Web 界面
1. 访问 `http://localhost:8080`
2. 勾选 "Use Auto Document Feeder (ADF)"
3. 设置页数（例如 50）
4. 点击 "Start Scan"
5. 扫描仪会以**最大速度**连续扫描

### API 调用
```bash
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "scanner_id": "your-scanner-id",
    "parameters": {
      "resolution": 300,
      "color_mode": "Grayscale",
      "format": "JPEG",
      "use_feeder": true,
      "page_count": 50
    }
  }'
```

## 高速扫描仪实际速度

| 扫描仪型号 | 速度 (ppm) | 50 页时间 | 100 页时间 |
|-----------|-----------|----------|-----------|
| Fujitsu fi-7160 | 60 | 50 秒 | 100 秒 |
| Fujitsu fi-7180 | 80 | 38 秒 | 75 秒 |
| Fujitsu fi-7260 | 60 | 50 秒 | 100 秒 |
| Fujitsu fi-7280 | 80 | 38 秒 | 75 秒 |
| Fujitsu fi-8170 | 70 | 43 秒 | 86 秒 |
| Canon DR-C225 II | 25 | 120 秒 | 240 秒 |
| Canon DR-G2140 | 140 | 21 秒 | 43 秒 |

*ppm = pages per minute（每分钟页数）*

## 进一步优化建议

### 1. 使用合适的分辨率

| 用途 | 分辨率 | 速度影响 |
|------|--------|---------|
| 快速归档 | 150-200 DPI | 最快 ⚡⚡⚡ |
| 标准文档 | 300 DPI | 快 ⚡⚡ |
| OCR 识别 | 300 DPI | 快 ⚡⚡ |
| 高质量归档 | 600 DPI | 慢 ⚡ |

### 2. 选择高效的颜色模式

| 模式 | 文件大小 | 扫描速度 | 适用场景 |
|------|---------|---------|---------|
| Black & White | 最小 | 最快 ⚡⚡⚡ | 纯文本文档 |
| Grayscale | 中等 | 快 ⚡⚡ | 大多数文档 |
| Color | 最大 | 慢 ⚡ | 彩色图表、照片 |

### 3. 使用 SSD 存储

扫描结果直接保存到 SSD 而不是 HDD：
- HDD 写入：50-100 MB/s
- SSD 写入：500-3500 MB/s

**影响：**
- 对于 300 DPI 灰度 JPEG（约 200 KB/页）
- HDD：2000 张/秒 理论上限
- SSD：17500 张/秒 理论上限
- **实际瓶颈是扫描仪速度，不是磁盘**

### 4. 批量处理

对于超大批量扫描（数百页）：
```python
import requests

# 分批扫描，每批 50 页
total_pages = 500
batch_size = 50
batches = total_pages // batch_size

for batch in range(batches):
    print(f"扫描批次 {batch + 1}/{batches}")

    input("放入下 50 页，按 Enter 继续...")

    response = requests.post('http://localhost:8080/api/v1/scan', json={
        'scanner_id': 'scanner-001',
        'parameters': {
            'resolution': 200,
            'color_mode': 'Grayscale',
            'format': 'JPEG',
            'use_feeder': True,
            'page_count': batch_size
        }
    })

    print(f"批次 {batch + 1} 完成")

print("全部完成！")
```

## 故障排除

### 问题：扫描仍然很慢

**可能原因：**
1. 扫描仪本身速度慢（检查型号规格）
2. 分辨率太高（降低到 300 DPI）
3. 使用彩色模式（改为灰度）
4. 扫描仪需要预热（第一次扫描后会变快）

**解决方案：**
```bash
# 查看扫描仪规格
curl http://localhost:8080/api/v1/scanners

# 使用优化的参数
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "scanner_id": "scanner-001",
    "parameters": {
      "resolution": 200,           # 降低分辨率
      "color_mode": "Grayscale",   # 灰度模式
      "format": "JPEG",            # JPEG 压缩
      "use_feeder": true,
      "page_count": 50
    }
  }'
```

### 问题：某些页面模糊或缺失

**可能原因：**
扫描仪进纸太快，导致某些页面没有充分扫描

**解决方案：**
如果确实需要在某些页面之间添加短暂延迟，可以通过配置文件设置：

```yaml
# config.yaml
scanner:
  adf:
    page_delay_ms: 50  # 每页之间延迟 50ms
```

**权衡：**
- 延迟越小 = 速度越快，但可能影响质量
- 延迟越大 = 质量越好，但速度变慢
- **建议：先尝试 0ms，如果有问题再逐步增加到 50ms**

## 总结

✅ **移除了 100ms 的人为延迟**
✅ **扫描仪现在以最大硬件速度运行**
✅ **性能提升 6.7% - 16.7%**
✅ **代码更简洁、更高效**

现在您的高速扫描仪可以真正发挥全部性能！🚀
