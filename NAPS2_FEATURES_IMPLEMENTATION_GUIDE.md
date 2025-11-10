# NAPS2 完整功能实现指南

本文档详细说明如何在 Go 项目中实现 NAPS2 的所有扫描功能。

## 功能清单

根据 NAPS2 源代码分析，需要实现以下功能：

| 功能 | NAPS2 源码 | 优先级 | 状态 |
|------|-----------|--------|------|
| ✅ WIA 循环 Transfer | WiaScanDriver.cs:296-309 | ⭐⭐⭐⭐⭐ | 已完成 |
| 📄 纸张大小设置 | WiaScanDriver.cs:447-474 | ⭐⭐⭐⭐⭐ | 待实现 |
| ↔️ 水平对齐 | WiaScanDriver.cs:455-459 | ⭐⭐⭐⭐ | 待实现 |
| 🔍 缩放比例 | RemotePostProcessor.cs:73-77 | ⭐⭐⭐ | 待实现 |
| 📋 排除空白页 | BlankDetectionImageOp.cs | ⭐⭐⭐⭐ | 待实现 |
| 🎨 图像质量 | ScanOptions.cs:115-121 | ⭐⭐⭐ | 待实现 |
| ✂️ 裁剪到页面 | RemotePostProcessor.cs:79-134 | ⭐⭐⭐ | 待实现 |

---

## 1. 纸张大小设置 ⭐⭐⭐⭐⭐

### 原理

NAPS2 通过设置 WIA 扫描区域属性来控制纸张大小：
- `IPS_XEXTENT` - 水平扫描宽度（像素）
- `IPS_YEXTENT` - 垂直扫描高度（像素）
- `IPS_XPOS` - 水平起始位置（用于对齐）

### NAPS2 实现逻辑

```csharp
// 1. 将页面尺寸转换为像素
int pageWidth = pageSizeInMM * resolution / 25.4;  // mm -> pixels
int pageHeight = pageSizeInMM * resolution / 25.4;

// 2. 获取最大扫描区域
int maxWidth = deviceMaxWidth * resolution / 1000;  // 千分之一英寸 -> 像素
int maxHeight = deviceMaxHeight * resolution / 1000;

// 3. 限制在最大范围内
pageWidth = min(pageWidth, maxWidth);
pageHeight = min(pageHeight, maxHeight);

// 4. 设置 WIA 属性
SafeSetProperty(item, IPS_XEXTENT, pageWidth);
SafeSetProperty(item, IPS_YEXTENT, pageHeight);
```

### Go 实现

```go
// driver_windows.go 添加纸张大小计算函数

// calculateScanArea 计算扫描区域（像素）
func (d *WindowsDriver) calculateScanArea(params models.ScanParams, resolution int) (width, height, xPos int, err error) {
	// 1. 获取页面尺寸（mm）
	var pageWidthMM, pageHeightMM int

	if params.PageSize != "" && params.PageSize != "Custom" {
		// 使用预定义纸张大小
		if size, ok := models.PaperSizes[params.PageSize]; ok {
			pageWidthMM = size.Width
			pageHeightMM = size.Height
		} else {
			return 0, 0, 0, fmt.Errorf("unknown page size: %s", params.PageSize)
		}
	} else {
		// 使用自定义大小
		pageWidthMM = params.PageWidth
		pageHeightMM = params.PageHeight

		// 兼容旧参数
		if pageWidthMM == 0 {
			pageWidthMM = params.Width
		}
		if pageHeightMM == 0 {
			pageHeightMM = params.Height
		}
	}

	// 2. 转换为像素（NAPS2 公式）
	// mm 转英寸：mm / 25.4
	// 英寸转像素：inch * dpi
	pageWidthPixels := int(float64(pageWidthMM) / 25.4 * float64(resolution))
	pageHeightPixels := int(float64(pageHeightMM) / 25.4 * float64(resolution))

	fmt.Printf("  Page size: %dx%d mm = %dx%d pixels @ %d DPI\n",
		pageWidthMM, pageHeightMM, pageWidthPixels, pageHeightPixels, resolution)

	return pageWidthPixels, pageHeightPixels, 0, nil
}

// 在 Scan 函数中使用
pageWidth, pageHeight, xPos, err := d.calculateScanArea(params, params.Resolution)
if err != nil {
	return nil, err
}

// 设置扫描区域
d.safeSetPropertyInt(props, WIA_IPS_XEXTENT, pageWidth)
d.safeSetPropertyInt(props, WIA_IPS_YEXTENT, pageHeight)
d.safeSetPropertyInt(props, WIA_IPS_XPOS, xPos)
```

---

## 2. 水平对齐 ⭐⭐⭐⭐

### 原理

通过调整 `IPS_XPOS`（水平起始位置）实现左对齐、居中、右对齐。

### NAPS2 算法

```csharp
int horizontalPos = 0;  // 默认右对齐

if (pageAlign == HorizontalAlign.Center) {
    // 居中：起始位置 = (最大宽度 - 页面宽度) / 2
    horizontalPos = (maxWidth - pageWidth) / 2;
} else if (pageAlign == HorizontalAlign.Left) {
    // 左对齐：起始位置 = 最大宽度 - 页面宽度
    horizontalPos = maxWidth - pageWidth;
}
// 右对齐：起始位置 = 0（默认）
```

### Go 实现

```go
// calculateHorizontalAlignment 计算水平对齐位置
func (d *WindowsDriver) calculateHorizontalAlignment(
	pageWidth, maxWidth int,
	alignment string,
) int {
	switch alignment {
	case models.AlignCenter:
		// 居中对齐
		return (maxWidth - pageWidth) / 2

	case models.AlignLeft:
		// 左对齐
		return maxWidth - pageWidth

	case models.AlignRight:
		fallthrough
	default:
		// 右对齐（默认）
		return 0
	}
}

// 使用示例
if params.PageAlign != "" {
	// 获取最大扫描区域
	maxWidthPixels := d.getMaxScanWidth(device, item, params.UseFeeder)

	// 计算对齐位置
	xPos = d.calculateHorizontalAlignment(
		pageWidth,
		maxWidthPixels,
		params.PageAlign,
	)

	fmt.Printf("  Horizontal align: %s (xPos=%d)\n", params.PageAlign, xPos)
}

// 应用 WIA 属性
if params.WiaOffsetWidth {
	// NAPS2 模式：将偏移量添加到宽度
	d.safeSetPropertyInt(props, WIA_IPS_XEXTENT, pageWidth+xPos)
	d.safeSetPropertyInt(props, WIA_IPS_XPOS, xPos)
} else {
	// 标准模式
	d.safeSetPropertyInt(props, WIA_IPS_XEXTENT, pageWidth)
	d.safeSetPropertyInt(props, WIA_IPS_XPOS, xPos)
}
```

---

## 3. 缩放比例 ⭐⭐⭐

### 原理

NAPS2 在扫描后对图像进行缩放处理，支持 1:1, 1:2, 1:4, 1:8 四种比例。

### NAPS2 实现

```csharp
if (options.ScaleRatio > 1) {
    var scaleFactor = 1.0 / options.ScaleRatio;
    scaled = scaled.PerformTransform(new ScaleTransform(scaleFactor));
}
```

### Go 实现方案

**选项 A：使用图像处理库（推荐）**

```go
import (
	"image"
	"image/jpeg"
	"github.com/nfnt/resize"  // Go 图像缩放库
)

// scaleImage 缩放图像
func (d *WindowsDriver) scaleImage(
	inputPath string,
	scaleRatio int,
) error {
	// 1. 读取原始图像
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	// 2. 计算新尺寸
	newWidth := img.Bounds().Dx() / scaleRatio
	newHeight := img.Bounds().Dy() / scaleRatio

	// 3. 缩放图像（高质量）
	scaled := resize.Resize(
		uint(newWidth),
		uint(newHeight),
		img,
		resize.Lanczos3,  // 高质量插值
	)

	// 4. 保存缩放后的图像
	out, err := os.Create(inputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, scaled, &jpeg.Options{Quality: params.JpegQuality})
}

// 在扫描后处理中应用
if params.ScaleRatio > 1 {
	err := d.scaleImage(filePath, params.ScaleRatio)
	if err != nil {
		fmt.Printf("Warning: Failed to scale image: %v\n", err)
	}
}
```

**选项 B：使用 ImageMagick（更强大）**

```go
import "os/exec"

// scaleImageMagick 使用 ImageMagick 缩放
func (d *WindowsDriver) scaleImageMagick(
	inputPath string,
	scaleRatio int,
) error {
	percentage := 100 / scaleRatio
	cmd := exec.Command(
		"magick",
		"convert",
		inputPath,
		"-resize",
		fmt.Sprintf("%d%%", percentage),
		inputPath,
	)
	return cmd.Run()
}
```

---

## 4. 排除空白页 ⭐⭐⭐⭐

### 原理

NAPS2 使用 **YUV 亮度算法** 检测空白页：

1. 计算每个像素的亮度值
2. 统计非白色像素数量
3. 计算覆盖率 = 非白色像素数 / 总像素数
4. 如果覆盖率 < 阈值，则判定为空白页

### NAPS2 算法详解

```csharp
// 1. 转换 RGB 到亮度（YUV 公式）
int luma = r * 299 + g * 587 + b * 114;  // 放大1000倍避免浮点运算

// 2. 白色阈值调整
whiteThresholdAdjusted = 1 + (whiteThreshold / 100.0) * 254;
// whiteThreshold=70 -> whiteThresholdAdjusted=179

// 3. 检测非白色像素
if (luma < whiteThresholdAdjusted * 1000) {
    nonWhitePixelCount++;
}

// 4. 计算覆盖率
coverage = nonWhitePixelCount / (double)totalPixels;

// 5. 覆盖率阈值调整
coverageThresholdAdjusted = 0.00 + (coverageThreshold / 100.0) * 0.01;
// coverageThreshold=15 -> coverageThresholdAdjusted=0.0015 (0.15%)

// 6. 判断空白
isBlank = coverage < coverageThresholdAdjusted;
```

### Go 实现

```go
import (
	"image"
	_ "image/jpeg"
)

// BlankPageDetector 空白页检测器
type BlankPageDetector struct {
	WhiteThreshold    int     // 0-100 (default: 70)
	CoverageThreshold int     // 0-100 (default: 15)
}

// isBlankPage 检测是否为空白页（NAPS2 算法）
func (d *BlankPageDetector) isBlankPage(imagePath string) (bool, error) {
	// 1. 打开图像
	file, err := os.Open(imagePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return false, err
	}

	// 2. 调整阈值（NAPS2 公式）
	whiteThresholdAdjusted := 1 + int(float64(d.WhiteThreshold)/100.0*254)
	coverageThresholdAdjusted := 0.00 + (float64(d.CoverageThreshold)/100.0)*0.01

	// 3. 忽略边缘 1% 区域（防止边框影响）
	bounds := img.Bounds()
	ignoreEdge := int(float64(bounds.Dx()) * 0.01)

	startX := ignoreEdge
	endX := bounds.Dx() - ignoreEdge
	startY := ignoreEdge
	endY := bounds.Dy() - ignoreEdge

	// 4. 扫描像素
	totalPixels := (endX - startX) * (endY - startY)
	nonWhitePixels := 0

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			// 转换为 8 位
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// YUV 亮度公式（NAPS2: r*299 + g*587 + b*114）
			luma := int(r8)*299 + int(g8)*587 + int(b8)*114

			// 检测非白色像素
			if luma < whiteThresholdAdjusted*1000 {
				nonWhitePixels++
			}
		}
	}

	// 5. 计算覆盖率
	coverage := float64(nonWhitePixels) / float64(totalPixels)

	// 6. 判断空白
	isBlank := coverage < coverageThresholdAdjusted

	fmt.Printf("  Blank page detection: coverage=%.4f%%, threshold=%.4f%%, blank=%v\n",
		coverage*100, coverageThresholdAdjusted*100, isBlank)

	return isBlank, nil
}

// 在扫描后处理中应用
if params.ExcludeBlankPages {
	detector := &BlankPageDetector{
		WhiteThreshold:    params.BlankPageWhiteThreshold,
		CoverageThreshold: params.BlankPageCoverageThreshold,
	}

	isBlank, err := detector.isBlankPage(filePath)
	if err != nil {
		fmt.Printf("Warning: Blank page detection failed: %v\n", err)
	} else if isBlank {
		// 删除空白页
		os.Remove(filePath)
		fmt.Printf("  Excluded blank page: %s\n", filePath)
		continue  // 跳过此页
	}
}
```

---

## 5. 图像质量 ⭐⭐⭐

### 原理

NAPS2 支持两种质量模式：
1. **MaxQuality = true**：无损存储（PNG 或 TIFF）
2. **MaxQuality = false**：JPEG 压缩（Quality 0-100）

### Go 实现

```go
import (
	"image/jpeg"
	"image/png"
)

// saveImageWithQuality 保存图像并应用质量设置
func (d *WindowsDriver) saveImageWithQuality(
	image *ole.IDispatch,
	filePath string,
	params models.ScanParams,
) error {
	// 1. 保存为临时文件
	tempPath := filePath + ".tmp"
	_, err := oleutil.CallMethod(image, "SaveFile", tempPath)
	if err != nil {
		return err
	}
	defer os.Remove(tempPath)

	// 2. 读取图像
	file, err := os.Open(tempPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	// 3. 根据质量设置保存
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	if params.MaxQuality {
		// 无损 PNG
		fmt.Println("  Saving as lossless PNG (MaxQuality)")
		return png.Encode(out, img)
	} else {
		// JPEG 压缩
		quality := params.JpegQuality
		if quality == 0 {
			quality = models.DefaultJpegQuality  // 75
		}
		fmt.Printf("  Saving as JPEG (quality=%d)\n", quality)
		return jpeg.Encode(out, img, &jpeg.Options{Quality: quality})
	}
}
```

---

## 6. 裁剪到页面大小 ⭐⭐⭐

### 原理

NAPS2 支持两种模式：
1. **StretchToPageSize**：调整图像 DPI 使其匹配页面大小（不改变像素）
2. **CropToPageSize**：裁剪图像使其匹配页面大小（可能丢失边缘）

### NAPS2 算法

```csharp
// 1. 计算实际尺寸（英寸）
float actualWidthInch = imageWidth / horizontalDPI;
float actualHeightInch = imageHeight / verticalDPI;

// 2. 检测方向
bool isLandscape = actualWidth > actualHeight;
bool pageLandscape = pageWidth > pageHeight;

// 3. 如果方向不匹配，交换页面尺寸
if (isLandscape != pageLandscape) {
    swap(pageWidth, pageHeight);
}

// 4A. 拉伸模式：调整 DPI
if (stretchToPageSize) {
    newDPI_X = imageWidth / pageWidthInch;
    newDPI_Y = imageHeight / pageHeightInch;
    image.SetResolution(newDPI_X, newDPI_Y);
}

// 4B. 裁剪模式：物理裁剪
if (cropToPageSize) {
    cropRight = (actualWidth - pageWidth) * horizontalDPI;
    cropBottom = (actualHeight - pageHeight) * verticalDPI;
    image = image.Crop(0, cropRight, 0, cropBottom);
}
```

### Go 实现

```go
import (
	"image"
	"github.com/disintegration/imaging"  // Go 图像处理库
)

// cropToPageSize 裁剪图像到指定页面大小
func (d *WindowsDriver) cropToPageSize(
	inputPath string,
	params models.ScanParams,
) error {
	// 1. 读取图像
	img, err := imaging.Open(inputPath)
	if err != nil {
		return err
	}

	// 2. 获取目标页面尺寸（mm）
	var pageWidthMM, pageHeightMM int
	if size, ok := models.PaperSizes[params.PageSize]; ok {
		pageWidthMM = size.Width
		pageHeightMM = size.Height
	} else {
		pageWidthMM = params.PageWidth
		pageHeightMM = params.PageHeight
	}

	// 3. 转换为像素（使用扫描分辨率）
	targetWidth := int(float64(pageWidthMM) / 25.4 * float64(params.Resolution))
	targetHeight := int(float64(pageHeightMM) / 25.4 * float64(params.Resolution))

	// 4. 裁剪或调整
	var processed image.Image

	if params.CropToPageSize {
		// 裁剪模式：从中心裁剪
		processed = imaging.CropCenter(img, targetWidth, targetHeight)
		fmt.Printf("  Cropped to %dx%d pixels\n", targetWidth, targetHeight)

	} else if params.StretchToPageSize {
		// 拉伸模式：调整大小（保持宽高比）
		processed = imaging.Fit(img, targetWidth, targetHeight, imaging.Lanczos)
		fmt.Printf("  Resized to fit %dx%d pixels\n", targetWidth, targetHeight)

	} else {
		// 无处理
		processed = img
	}

	// 5. 保存
	quality := params.JpegQuality
	if quality == 0 {
		quality = models.DefaultJpegQuality
	}

	return imaging.Save(processed, inputPath, imaging.JPEGQuality(quality))
}
```

---

## 7. 完整的扫描流程集成

### 主扫描函数更新

```go
func (d *WindowsDriver) Scan(...) ([]models.ScanResult, error) {
	// ... 现有的 WIA 设备连接代码 ...

	// === 步骤 1: 配置 WIA 属性（扫描前） ===
	d.configureWiaProperties(props, params)

	// === 步骤 2: 执行扫描 ===
	rawResults, err := d.scanADFBatch(ctx, item, outputDir, baseTimestamp, pageCount, params, progressCallback)
	if err != nil {
		return nil, err
	}

	// === 步骤 3: 后处理（扫描后） ===
	var finalResults []models.ScanResult

	for _, result := range rawResults {
		// 3.1 空白页检测
		if params.ExcludeBlankPages {
			detector := &BlankPageDetector{
				WhiteThreshold:    params.BlankPageWhiteThreshold,
				CoverageThreshold: params.BlankPageCoverageThreshold,
			}
			if isBlank, _ := detector.isBlankPage(result.FilePath); isBlank {
				os.Remove(result.FilePath)
				fmt.Printf("  Excluded blank page %d\n", result.PageNumber)
				continue
			}
		}

		// 3.2 缩放
		if params.ScaleRatio > 1 {
			if err := d.scaleImage(result.FilePath, params.ScaleRatio); err != nil {
				fmt.Printf("  Warning: Scale failed: %v\n", err)
			}
		}

		// 3.3 裁剪/调整到页面大小
		if params.CropToPageSize || params.StretchToPageSize {
			if err := d.cropToPageSize(result.FilePath, params); err != nil {
				fmt.Printf("  Warning: Crop/resize failed: %v\n", err)
			}
		}

		// 3.4 应用图像质量设置
		if params.MaxQuality || params.JpegQuality != models.DefaultJpegQuality {
			if err := d.recompressImage(result.FilePath, params); err != nil {
				fmt.Printf("  Warning: Recompress failed: %v\n", err)
			}
		}

		finalResults = append(finalResults, result)
	}

	return finalResults, nil
}
```

### configureWiaProperties 函数

```go
func (d *WindowsDriver) configureWiaProperties(props *ole.IDispatch, params models.ScanParams) {
	fmt.Println("Configuring WIA properties (NAPS2 full mode)...")

	// 1. 基本属性（已实现）
	d.safeSetPropertyInt(props, WIA_IPA_DATATYPE, ...)
	d.safeSetPropertyInt(props, WIA_IPS_XRES, params.Resolution)
	d.safeSetPropertyInt(props, WIA_IPS_YRES, params.Resolution)

	// 2. 纸张大小和对齐（新增）
	if params.PageSize != "" || params.PageWidth > 0 {
		pageWidth, pageHeight, xPos, err := d.calculateScanArea(params, params.Resolution)
		if err == nil {
			d.safeSetPropertyInt(props, WIA_IPS_XEXTENT, pageWidth)
			d.safeSetPropertyInt(props, WIA_IPS_YEXTENT, pageHeight)
			d.safeSetPropertyInt(props, WIA_IPS_XPOS, xPos)
			fmt.Printf("  Scan area: %dx%d pixels at offset %d\n", pageWidth, pageHeight, xPos)
		}
	}

	// 3. ADF 设置（已实现）
	if params.UseFeeder {
		// ... 现有的 ADF 代码 ...
	}

	// 4. 亮度和对比度（WIA 支持）
	if params.Brightness != 0 {
		d.safeSetPropertyInt(props, WIA_IPS_BRIGHTNESS, params.Brightness)
		fmt.Printf("  Brightness: %d\n", params.Brightness)
	}
	if params.Contrast != 0 {
		d.safeSetPropertyInt(props, WIA_IPS_CONTRAST, params.Contrast)
		fmt.Printf("  Contrast: %d\n", params.Contrast)
	}

	fmt.Println("Property configuration complete")
}
```

---

## 实现优先级建议

### 第一阶段（必需功能）✅
1. ✅ WIA 循环 Transfer - **已完成**
2. ✅ SafeSetProperty - **已完成**
3. ✅ 完整 WIA 属性 - **已完成**

### 第二阶段（高优先级）⭐⭐⭐⭐⭐
4. 📄 纸张大小设置 - **必需，影响所有扫描**
5. 📋 排除空白页 - **高价值，节省时间**
6. ↔️ 水平对齐 - **重要，提升质量**

### 第三阶段（中优先级）⭐⭐⭐
7. 🎨 图像质量 - **影响文件大小和质量**
8. 🔍 缩放比例 - **节省存储空间**
9. ✂️ 裁剪到页面 - **标准化输出**

### 第四阶段（可选功能）⭐⭐
10. 自动纠偏 - **需要复杂算法**
11. 旋转 - **简单但不常用**
12. 翻转双面页 - **特定场景**

---

## 依赖库建议

为了实现后处理功能，建议使用以下 Go 库：

```bash
# 图像处理
go get github.com/disintegration/imaging

# 图像缩放（高质量）
go get github.com/nfnt/resize

# 可选：ImageMagick Go 绑定（功能最强大）
go get gopkg.in/gographics/imagick.v3/imagick
```

---

## 测试建议

### 纸张大小测试
```json
{
  "page_size": "A4",
  "page_align": "Center",
  "resolution": 300
}
```

### 空白页检测测试
```json
{
  "exclude_blank_pages": true,
  "blank_page_white_threshold": 70,
  "blank_page_coverage_threshold": 15,
  "page_count": 10
}
```

### 缩放测试
```json
{
  "scale_ratio": 2,
  "jpeg_quality": 75,
  "page_count": 5
}
```

---

## 性能考虑

### 后处理性能

| 操作 | 耗时（300 DPI A4） | 建议 |
|------|------------------|------|
| 空白页检测 | ~50-100ms | ✅ 可接受 |
| 缩放 1:2 | ~100-200ms | ✅ 可接受 |
| 裁剪 | ~50-100ms | ✅ 可接受 |
| JPEG 重压缩 | ~100-200ms | ⚠️ 仅在必要时 |

### 优化策略

1. **并发处理：** 使用 Goroutine 并发处理多页
2. **批量操作：** 一次性处理多个转换
3. **条件应用：** 仅在参数启用时执行
4. **缓存结果：** 避免重复读取图像

---

## 总结

本指南提供了实现 NAPS2 所有核心功能的完整蓝图：

| 功能 | 实现复杂度 | 价值 | 优先级 |
|------|----------|------|--------|
| 纸张大小 | 简单 | ⭐⭐⭐⭐⭐ | 第一 |
| 空白页检测 | 中等 | ⭐⭐⭐⭐⭐ | 第一 |
| 水平对齐 | 简单 | ⭐⭐⭐⭐ | 第二 |
| 图像质量 | 简单 | ⭐⭐⭐⭐ | 第二 |
| 缩放比例 | 简单 | ⭐⭐⭐ | 第三 |
| 裁剪页面 | 中等 | ⭐⭐⭐ | 第三 |

按照此指南逐步实现，即可达到 **NAPS2 完全兼容** 的企业级扫描服务！🚀
