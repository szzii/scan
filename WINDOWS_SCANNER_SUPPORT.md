# Windows 扫描仪支持文档

## 概述

scanserver 在 Windows 平台上支持两种扫描协议：

1. **WIA (Windows Image Acquisition)** - Windows 原生现代扫描协议
2. **TWAIN** - 跨平台传统扫描协议（兼容性更好）

系统会自动检测并使用可用的协议，优先使用 WIA，如果 WIA 不可用则回退到 TWAIN。

## WIA (Windows Image Acquisition) 支持

### 什么是 WIA？

WIA 是 Windows 内置的扫描和成像 API，从 Windows ME/XP 开始引入。现代的 Windows 扫描仪通常都支持 WIA。

### WIA 的优势

- ✅ Windows 原生支持，无需额外驱动
- ✅ 现代化的 API 设计
- ✅ 支持自动设备发现
- ✅ 更好的系统集成

### WIA 支持的功能

- 自动检测已连接的 WIA 兼容扫描仪
- 设置扫描分辨率 (75-1200 DPI)
- 选择颜色模式（彩色、灰度、黑白）
- 支持多种图片格式（JPEG, PNG, TIFF, BMP）
- 实时扫描进度反馈

### 使用要求

1. Windows 操作系统（Windows XP 或更高版本）
2. 扫描仪已正确安装并在"设备和打印机"中可见
3. 扫描仪驱动支持 WIA 协议

### 检查扫描仪是否支持 WIA

1. 打开 "设备和打印机" (Win+R -> `control printers`)
2. 找到你的扫描仪设备
3. 右键点击 -> "属性"
4. 如果有 "扫描配置文件" 或 "WIA" 相关选项，则支持 WIA

或者使用命令行：
```cmd
wiatest
```

## TWAIN 协议支持

### 什么是 TWAIN？

TWAIN 是一个跨平台的扫描仪和数字相机接口标准，由 TWAIN 工作组开发。它是业界标准，几乎所有扫描仪都支持。

### TWAIN 的优势

- ✅ 广泛的设备兼容性
- ✅ 跨平台支持（Windows, macOS, Linux）
- ✅ 成熟稳定的协议
- ✅ 支持高级功能（自动送纸器、双面扫描等）

### TWAIN 使用要求

1. 安装 TWAIN DSM (Data Source Manager)
   - Windows 32位：需要 `TWAIN_32.dll` 和 `TWAINDSM.dll`
   - Windows 64位：需要 `TWAINDSM.dll`
2. 安装扫描仪的 TWAIN 驱动程序

### 安装 TWAIN DSM

大多数扫描仪驱动会自动安装 TWAIN DSM。如果没有：

1. 访问 [TWAIN 官网](https://www.twain.org/)
2. 下载并安装 TWAIN DSM

或使用 scanserver 自带的 TWAIN 检测：
- scanserver 会自动检测系统是否安装了 TWAIN DSM
- 如果未找到会显示错误消息并提供安装指导

## 如何使用

### 启动服务

```cmd
scanserver.exe
```

或指定配置：

```cmd
scanserver.exe -host localhost -port 8080
```

### 检查扫描仪是否被检测到

1. 打开浏览器访问: `http://localhost:8080`
2. 查看 "Available Scanners" 列表
3. 如果没有扫描仪显示，检查：
   - 扫描仪是否开机并连接到电脑
   - 扫描仪驱动是否正确安装
   - Windows 设备管理器中是否能看到扫描仪

### API 调用示例

列出所有扫描仪：
```bash
curl http://localhost:8080/api/v1/scanners
```

创建扫描任务：
```bash
curl -X POST http://localhost:8080/api/v1/scan \
  -H "Content-Type: application/json" \
  -d '{
    "scanner_id": "your-scanner-id",
    "parameters": {
      "resolution": 300,
      "color_mode": "Color",
      "format": "JPEG",
      "width": 210,
      "height": 297
    }
  }'
```

## 故障排除

### 找不到扫描仪

**问题**: scanserver 启动后找不到扫描仪

**解决方案**:

1. **检查扫描仪连接**
   - 确保扫描仪开机
   - 检查 USB 连接
   - 重新插拔 USB 线

2. **检查驱动安装**
   ```cmd
   # 打开设备管理器
   devmgmt.msc
   ```
   - 查找 "成像设备" 或 "扫描仪" 分类
   - 确保没有黄色感叹号
   - 如有问题，重新安装驱动

3. **测试 WIA 连接**
   ```cmd
   # 运行 Windows 扫描向导
   wiaacmgr
   ```
   如果 Windows 扫描向导能看到扫描仪，说明 WIA 工作正常

4. **测试 TWAIN 连接**
   - 打开任何支持 TWAIN 的应用（如 Paint.NET, GIMP）
   - 尝试从扫描仪导入图像
   - 如果能看到扫描仪，说明 TWAIN 工作正常

### WIA 初始化失败

**错误消息**: `failed to create WIA DeviceManager: ...`

**解决方案**:

1. 检查 Windows 组件
   ```cmd
   # 以管理员身份运行
   sfc /scannow
   ```

2. 重新注册 WIA COM 对象
   ```cmd
   # 以管理员身份运行
   regsvr32 wiashext.dll
   regsvr32 wiaservc.dll
   ```

3. 重启 Windows Image Acquisition 服务
   ```cmd
   net stop stisvc
   net start stisvc
   ```

### TWAIN DSM 未找到

**错误消息**: `TWAIN DSM not found (TWAINDSM.dll)`

**解决方案**:

1. 下载并安装 TWAIN DSM:
   - 访问 https://www.twain.org/downloads/
   - 下载适合你系统的版本（32位或64位）
   - 安装后重启 scanserver

2. 或者从扫描仪制造商网站下载完整驱动包

### COM 初始化错误

**错误消息**: `failed to initialize COM: ...`

**解决方案**:

1. 以管理员权限运行 scanserver
   ```cmd
   # 右键点击 scanserver.exe -> "以管理员身份运行"
   ```

2. 检查系统权限设置
   ```cmd
   dcomcnfg
   ```
   - 确保当前用户有权限访问 COM 对象

## 支持的扫描仪

理论上，scanserver 支持所有兼容 WIA 或 TWAIN 的扫描仪，包括但不限于：

### 主流品牌
- ✅ **HP** - LaserJet, OfficeJet, DeskJet 系列
- ✅ **Canon** - CanoScan, imageFORMULA 系列
- ✅ **Epson** - Perfection, WorkForce 系列
- ✅ **Brother** - MFC, DCP 系列
- ✅ **Fujitsu** - ScanSnap, fi 系列
- ✅ **Xerox** - WorkCentre, VersaLink 系列
- ✅ **Samsung** - SCX, M 系列

### 多功能一体机 (MFP)
大多数带扫描功能的多功能打印机都支持 WIA 和 TWAIN

## 技术细节

### WIA 实现

scanserver 使用 `go-ole` 库通过 COM 接口调用 Windows WIA API：

1. 初始化 COM 环境
2. 创建 WIA DeviceManager 对象
3. 枚举所有 WIA 设备
4. 筛选扫描仪类型设备
5. 读取设备属性（名称、制造商、ID）
6. 连接到选定的扫描仪
7. 设置扫描参数（分辨率、颜色模式等）
8. 执行扫描并保存图像

### TWAIN 实现

scanserver 通过 Windows 系统调用访问 TWAIN DSM：

1. 加载 TWAINDSM.dll
2. 获取 DSM_Entry 函数指针
3. 初始化应用程序标识
4. 打开 TWAIN DSM
5. 枚举数据源（扫描仪）
6. 选择并打开数据源
7. 设置扫描参数
8. 执行图像传输
9. 保存扫描结果

## 性能优化建议

1. **使用合适的分辨率**
   - 普通文档：150-200 DPI
   - 高质量文档：300 DPI
   - 照片：300-600 DPI
   - OCR 文字识别：300 DPI

2. **选择合适的颜色模式**
   - 彩色照片：Color
   - 普通文档：Grayscale
   - 文本文档：BlackAndWhite（最快）

3. **批量扫描**
   - 使用 `/api/v1/scan/batch` 端点
   - 启用自动送纸器（如果支持）

## 开发和调试

### 启用详细日志

设置环境变量：
```cmd
set GIN_MODE=debug
scanserver.exe
```

### 查看 COM 对象

使用 OleView 工具查看 WIA COM 对象：
```cmd
oleview.exe
```

查找 `WIA.DeviceManager` 类

## 更新日志

### v1.0.0
- ✅ 实现 WIA 协议支持
- ✅ 实现 TWAIN 协议支持
- ✅ 自动协议检测和回退
- ✅ 支持多种图像格式
- ✅ 实时进度反馈
- ✅ Web 界面图片预览

## 相关资源

- [WIA 官方文档](https://docs.microsoft.com/en-us/windows/win32/wia/)
- [TWAIN 官网](https://www.twain.org/)
- [go-ole 库](https://github.com/go-ole/go-ole)

## 技术支持

如有问题，请检查：

1. 日志输出中的错误信息
2. Windows 事件查看器中的相关错误
3. 扫描仪驱动是否为最新版本

如需帮助，请提供：
- Windows 版本
- 扫描仪型号
- 错误日志
- scanserver 版本
