# 前端功能说明

## 📱 Web 控制台已更新！

前端控制台现在包含所有 NAPS2 高级功能！

## 🎨 新增功能界面

访问 Web 控制台（默认：http://localhost:8080），你会看到：

### 基本扫描设置
- **Scanner** - 选择扫描器
- **Resolution** - 分辨率（150/300/600/1200 DPI）
- **Color Mode** - 颜色模式（彩色/灰度/黑白）
- **Format** - 输出格式（PDF/JPEG/PNG/TIFF）
- **Pages** - 页数
- **Use ADF** - 使用自动进纸器
- **Duplex** - 双面扫描

### 🆕 NAPS2 高级功能（点击展开）

点击 **"🎨 NAPS2 Advanced Features"** 展开高级设置：

#### 1. 纸张大小和对齐
- **Paper Size** - 选择纸张大小
  - Letter (8.5" x 11")
  - Legal (8.5" x 14")
  - A4 (210 x 297 mm) ⭐ 默认
  - A3 (297 x 420 mm)
  - A5 (148 x 210 mm)
  - B4 (250 x 353 mm)
  - B5 (176 x 250 mm)
  - A6 (105 x 148 mm)

- **Page Alignment** - 页面对齐方式
  - Left - 左对齐
  - Center - 居中对齐
  - Right - 右对齐

#### 2. 图像处理
- **Scale Ratio** - 缩放比例
  - 1:1 (100%) - 原始尺寸 ⭐ 默认
  - 1:2 (50%) - 缩小一半
  - 1:4 (25%) - 缩小到 1/4
  - 1:8 (12.5%) - 缩小到 1/8

- **JPEG Quality** - JPEG 质量（0-100，默认 75）

#### 3. 空白页检测
- **Exclude Blank Pages** - 排除空白页
  - 勾选后会显示额外设置：
    - **White Threshold** - 白色阈值（0-100，默认 70）
    - **Coverage Threshold** - 覆盖率阈值（0-100，默认 15）

#### 4. 图像质量和裁剪
- **Max Quality** - 最大质量（无损 PNG）
- **Crop to Page Size** - 裁剪到页面大小
- **Stretch to Page Size** - 拉伸到页面大小

## 📝 使用示例

### 示例 1：普通 A4 扫描
1. 选择扫描器
2. 勾选 "Use ADF"
3. 保持默认设置
4. 点击 "Start Scan"

### 示例 2：高质量彩色扫描（排除空白页）
1. 选择扫描器
2. Resolution: 600 DPI
3. Color Mode: Color
4. 勾选 "Use ADF"
5. 展开 "NAPS2 Advanced Features"
6. Paper Size: A4
7. 勾选 "Exclude Blank Pages"
8. 勾选 "Max Quality"
9. 点击 "Start Scan"

### 示例 3：节省空间的批量扫描
1. 选择扫描器
2. Resolution: 300 DPI
3. Color Mode: Grayscale
4. 勾选 "Use ADF"
5. Pages: 50
6. 展开 "NAPS2 Advanced Features"
7. Scale Ratio: 1:2 (50%)
8. JPEG Quality: 60
9. 点击 "Start Scan"

### 示例 4：精准裁剪 A4 文档
1. 选择扫描器
2. 勾选 "Use ADF"
3. 展开 "NAPS2 Advanced Features"
4. Paper Size: A4
5. Page Alignment: Center
6. 勾选 "Crop to Page Size"
7. 点击 "Start Scan"

## 🖼️ 界面截图说明

### 基本设置区域
```
┌─────────────────────────────────┐
│ Scanner: [下拉选择]              │
│ Resolution: [300] DPI            │
│ Color Mode: [Color]              │
│ Format: [PDF]                    │
│ Pages: [1]                       │
│ ☑ Use ADF                        │
│ ☐ Duplex                         │
└─────────────────────────────────┘
```

### NAPS2 高级功能区域（可折叠）
```
┌─────────────────────────────────┐
│ 🎨 NAPS2 Advanced Features ▼    │
├─────────────────────────────────┤
│ Paper Size: [A4 ▼]              │
│ Page Alignment: [Default ▼]     │
│                                  │
│ Scale Ratio: [1:1 ▼]            │
│ JPEG Quality: [75]               │
│                                  │
│ ☐ Exclude Blank Pages           │
│ ☐ Max Quality                   │
│                                  │
│ ☐ Crop to Page Size             │
│ ☐ Stretch to Page Size          │
└─────────────────────────────────┘
```

## 🔄 实时功能

### 1. 空白页设置自动显示
当勾选 "Exclude Blank Pages" 时，会自动显示：
- White Threshold 设置
- Coverage Threshold 设置

### 2. ADF 模式提示
当勾选 "Use ADF" 时，会显示蓝色提示框：
```
📄 ADF Mode: Place multiple pages in the document feeder.
The scanner will automatically scan all pages continuously.
```

### 3. 实时进度显示
扫描过程中会显示：
- 进度条
- 百分比
- 页面预览（完成后）

## 🎯 功能对应关系

| Web 界面选项 | API 参数 | NAPS2 功能 |
|-------------|---------|-----------|
| Paper Size | page_size | 纸张大小设置 |
| Page Alignment | page_align | 水平对齐 |
| Scale Ratio | scale_ratio | 缩放比例 |
| JPEG Quality | jpeg_quality | JPEG 质量 |
| Exclude Blank Pages | exclude_blank_pages | 排除空白页 |
| White Threshold | blank_page_white_threshold | 白色阈值 |
| Coverage Threshold | blank_page_coverage_threshold | 覆盖率阈值 |
| Max Quality | max_quality | 无损质量 |
| Crop to Page Size | crop_to_page_size | 裁剪到页面 |
| Stretch to Page Size | stretch_to_page_size | 拉伸到页面 |

## 💡 提示和技巧

### 提示 1：内存和存储优化
- 使用 Scale Ratio 1:2 可将文件大小减少约 75%
- 降低 JPEG Quality 到 60-70 可进一步减小文件
- 适合大量文档归档场景

### 提示 2：最佳质量设置
- Resolution: 600 DPI
- Color Mode: Color
- Max Quality: ✓
- Paper Size: A4
- Crop to Page Size: ✓
- 适合重要文档和照片扫描

### 提示 3：快速批量扫描
- Resolution: 300 DPI
- Color Mode: Grayscale
- Use ADF: ✓
- Exclude Blank Pages: ✓
- Scale Ratio: 1:2
- JPEG Quality: 70
- 适合日常办公文档

### 提示 4：OCR 友好设置
- Resolution: 300 DPI
- Color Mode: Grayscale 或 BlackAndWhite
- Paper Size: A4
- Crop to Page Size: ✓
- 适合需要 OCR 识别的文档

## 🚀 启动服务

```bash
# 编译（如果需要）
make build-all

# 运行服务
./build/scanserver

# 或在 Windows 上
./build/scanserver.exe
```

服务启动后，访问：
- **Web 控制台**: http://localhost:8080
- **API 端点**: http://localhost:8080/api/v1
- **WebSocket**: ws://localhost:8080/ws

## 🔗 相关文档

- [CHANGELOG.md](CHANGELOG.md) - 完整更新日志
- [FEATURES_SUMMARY.md](FEATURES_SUMMARY.md) - 功能摘要
- [NAPS2_FEATURES_IMPLEMENTATION_GUIDE.md](NAPS2_FEATURES_IMPLEMENTATION_GUIDE.md) - 实现指南

---

**版本**: v1.0.5
**更新时间**: 2025-11-10
**功能完整度**: 100% ✅

所有 NAPS2 功能现在都可以通过 Web 控制台访问！🎉
