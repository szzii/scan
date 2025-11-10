# 自动送纸器（ADF）批量扫描指南

## 功能概述

scanserver 支持使用扫描仪的自动送纸器（Automatic Document Feeder, ADF）进行批量扫描。只需一次配置，扫描仪会自动连续扫描多页文档，无需人工干预。

## 什么是 ADF？

ADF（自动送纸器）是高速扫描仪的标准配置，允许：
- ✅ 一次性放入多页文档（通常 20-100 页）
- ✅ 扫描仪自动逐页进纸和扫描
- ✅ 无需人工翻页或放置每一页
- ✅ 支持单面或双面（Duplex）扫描

## 使用方法

### Web 界面

1. **打开控制面板**
   - 访问 `http://localhost:8080`

2. **配置扫描参数**
   - 选择扫描仪
   - 设置分辨率（推荐 300 DPI）
   - 选择颜色模式
   - 选择输出格式

3. **启用 ADF 模式**
   - ✅ 勾选 "Use Auto Document Feeder (ADF)"
   - 设置要扫描的页数（例如：10 页）
   - 如果支持，可勾选 "Duplex Scanning" 进行双面扫描

4. **开始扫描**
   - 将多页文档放入扫描仪的送纸器
   - 点击 "Start Scan" 按钮
   - 扫描仪会自动连续扫描所有页面

### API 调用

```bash
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "scanner_id": "scanner-001",
    "parameters": {
      "resolution": 300,
      "color_mode": "Grayscale",
      "format": "PDF",
      "width": 210,
      "height": 297,
      "brightness": 0,
      "contrast": 0,
      "use_feeder": true,
      "use_duplex": false,
      "page_count": 10
    }
  }'
```

**参数说明：**
- `use_feeder: true` - 启用自动送纸器
- `use_duplex: true` - 双面扫描（如果支持）
- `page_count: 10` - 要扫描的页数

## 扫描模式对比

| 模式 | 人工操作 | 速度 | 适用场景 |
|------|---------|------|----------|
| **平板扫描** | 每页需手动放置 | 慢 | 单页、书籍、照片 |
| **ADF 单面** | 无需人工干预 | 快 | 多页文档、合同 |
| **ADF 双面** | 无需人工干预 | 最快 | 双面文档、发票 |

## 实际使用示例

### 示例 1：扫描 20 页合同

```javascript
const params = {
    scanner_id: "scanner-001",
    parameters: {
        resolution: 300,           // 适合 OCR 识别
        color_mode: "Grayscale",   // 节省空间
        format: "PDF",             // 便于分享
        use_feeder: true,          // 使用 ADF
        use_duplex: false,         // 单面
        page_count: 20             // 20 页
    }
};

fetch('/api/v1/scan', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(params)
});
```

### 示例 2：扫描 50 页双面发票

```javascript
const params = {
    scanner_id: "scanner-001",
    parameters: {
        resolution: 200,           // 快速扫描
        color_mode: "Color",       // 保留颜色信息
        format: "JPEG",            // 每页单独文件
        use_feeder: true,          // 使用 ADF
        use_duplex: true,          // 双面扫描
        page_count: 50             // 50 张纸 = 100 页
    }
};
```

**注意**：双面扫描时，`page_count` 指的是纸张数量，实际扫描页数会是 2 倍。

## 批量扫描端点

scanserver 还提供了专门的批量扫描端点：

```bash
curl -X POST http://localhost:8080/api/v1/scan/batch \
  -H "Content-Type: application/json" \
  -d '{
    "scanner_id": "scanner-001",
    "parameters": {
      "resolution": 300,
      "color_mode": "Grayscale",
      "format": "PDF",
      "use_feeder": true,
      "page_count": 10
    },
    "batch_count": 5
  }'
```

这会创建 5 个独立的扫描任务，每个任务扫描 10 页，适合需要多次扫描不同文档的场景。

## 性能优化建议

### 1. 选择合适的分辨率

| 用途 | 推荐分辨率 | 说明 |
|------|----------|------|
| 普通归档 | 150-200 DPI | 快速，文件小 |
| 标准文档 | 300 DPI | 平衡质量和速度 |
| OCR 识别 | 300 DPI | 最佳识别率 |
| 高质量归档 | 600 DPI | 高清，文件大 |

### 2. 颜色模式选择

- **Black & White (黑白)** - 最快，最小文件，适合纯文本
- **Grayscale (灰度)** - 平衡，适合大多数文档
- **Color (彩色)** - 最慢，最大文件，适合彩色图表

### 3. 输出格式

- **PDF** - 多页文档合并为一个文件，便于分享
- **JPEG** - 每页单独文件，通用格式
- **TIFF** - 高质量，支持压缩，适合归档
- **PNG** - 无损压缩，适合文字清晰度要求高的场景

## 故障排除

### 问题 1：ADF 不工作

**症状**：勾选 ADF 后仍然只扫描一页

**可能原因**：
1. 扫描仪不支持 ADF
2. 驱动未正确识别 ADF
3. 扫描仪设置问题

**解决方案**：
1. 检查扫描仪是否有送纸器
2. 更新扫描仪驱动
3. 在扫描仪控制面板中启用 ADF

### 问题 2：卡纸或进纸错误

**症状**：扫描过程中停止，显示错误

**解决方案**：
1. 打开扫描仪取出卡纸
2. 检查纸张是否整齐放置
3. 减少送纸器中的纸张数量
4. 使用质量更好的纸张

### 问题 3：页数不符

**症状**：设置 10 页但只扫描了 5 页

**可能原因**：
1. 送纸器中纸张不足
2. 多张纸一起进入
3. 扫描仪传感器问题

**解决方案**：
1. 确保放入足够的纸张
2. 将纸张扇开再放入
3. 清洁送纸器滚轮

### 问题 4：双面扫描顺序错乱

**症状**：双面扫描时页面顺序不对

**解决方案**：
1. 检查扫描仪双面扫描模式设置
2. 根据扫描仪型号调整纸张放置方向
3. 使用单面扫描后手动合并

## 高级功能

### 1. 实时进度监控

使用 WebSocket 监控扫描进度：

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);

    if (message.type === 'job_status') {
        const job = message.payload;
        console.log(`扫描进度: ${job.progress}%`);
        console.log(`已完成: ${job.results.length} 页`);
    }
};
```

### 2. 自动化脚本

批量处理多个文档：

```bash
#!/bin/bash

# 扫描 5 份不同的文档，每份 10 页
for i in {1..5}; do
    echo "请放入第 $i 份文档（10 页）"
    read -p "按 Enter 开始扫描..."

    curl -X POST http://localhost:8080/api/v1/scan \
      -H "Content-Type: application/json" \
      -d "{
        \"scanner_id\": \"scanner-001\",
        \"parameters\": {
          \"resolution\": 300,
          \"color_mode\": \"Grayscale\",
          \"format\": \"PDF\",
          \"use_feeder\": true,
          \"page_count\": 10
        }
      }"

    echo "第 $i 份文档扫描完成"
    sleep 2
done

echo "所有文档扫描完成！"
```

### 3. Python 自动化

```python
import requests
import time

def scan_document(scanner_id, page_count, description):
    """扫描一份文档"""
    print(f"开始扫描: {description} ({page_count} 页)")

    response = requests.post('http://localhost:8080/api/v1/scan', json={
        'scanner_id': scanner_id,
        'parameters': {
            'resolution': 300,
            'color_mode': 'Grayscale',
            'format': 'PDF',
            'use_feeder': True,
            'page_count': page_count,
            'use_duplex': False
        }
    })

    if response.ok:
        job = response.json()
        print(f"任务已创建: {job['id']}")
        return job['id']
    else:
        print(f"扫描失败: {response.text}")
        return None

# 使用示例
documents = [
    {"pages": 15, "name": "合同 A"},
    {"pages": 20, "name": "发票批次 1"},
    {"pages": 10, "name": "证明文件"}
]

for doc in documents:
    input(f"\n请放入: {doc['name']} ({doc['pages']} 页) 按 Enter 继续...")
    scan_document("scanner-001", doc['pages'], doc['name'])
    time.sleep(2)

print("\n所有文档扫描完成！")
```

## 最佳实践

### 1. 文档准备

- ✅ 移除钉书钉和回形针
- ✅ 确保纸张干净、平整
- ✅ 将纸张扇开，避免粘连
- ✅ 按照正确方向放置

### 2. 扫描设置

- ✅ 首次扫描测试 1-2 页验证设置
- ✅ 根据文档类型选择合适参数
- ✅ 批量扫描前清洁扫描仪
- ✅ 定期维护扫描仪滚轮

### 3. 文件管理

- ✅ 使用描述性文件名
- ✅ 按日期或类别组织文件
- ✅ 定期备份重要扫描文件
- ✅ 考虑使用 OCR 软件进行文字识别

## 支持的扫描仪

大多数商用扫描仪都支持 ADF 功能，包括：

### 高速文档扫描仪
- **Fujitsu ScanSnap** 系列 - 最高 40 ppm
- **Fujitsu fi** 系列 - 专业级，最高 100 ppm
- **Canon imageFORMULA** 系列 - 企业级
- **Epson WorkForce** 系列 - 办公级

### 多功能一体机
- **HP OfficeJet Pro** 系列 - 带 50 页 ADF
- **Brother MFC** 系列 - 带 35-50 页 ADF
- **Canon PIXMA** 系列 - 带 20-50 页 ADF

## 总结

ADF 批量扫描功能让 scanserver 成为高效的文档数字化工具：

✅ **自动化** - 无需人工翻页，节省时间
✅ **高效** - 一次扫描多页，提高效率
✅ **灵活** - 支持单面/双面，多种格式
✅ **可靠** - 实时进度反馈，错误处理

配合高速扫描仪使用，每分钟可扫描 40-100 页，大幅提升办公效率！
