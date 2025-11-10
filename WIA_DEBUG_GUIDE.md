# WIA 调试指南

## 当前问题分析

您遇到的错误 `failed to connect to scanner: 找不到成员。()` 表明 WIA COM 接口调用存在问题。

## 已修复的问题

1. ✅ **连接方法调用错误** - 修正了从 DeviceInfo 到 Device 的连接流程
2. ✅ **属性读取方式** - 改进了读取设备名称和制造商的方法

## 测试步骤

### 1. 验证扫描仪在 Windows 中可见

```cmd
# 方法1: 使用 Windows 扫描向导
wiaacmgr

# 方法2: 使用 WIA 测试工具（如果已安装 Windows SDK）
wiatest
```

### 2. 使用 PowerShell 检查 WIA 设备

```powershell
# 创建 WIA DeviceManager
$deviceManager = New-Object -ComObject WIA.DeviceManager

# 列出所有设备
$deviceManager.DeviceInfos | ForEach-Object {
    Write-Host "Device ID: $($_.DeviceID)"
    Write-Host "Type: $($_.Type)"

    # 遍历所有属性
    $_.Properties | ForEach-Object {
        Write-Host "  $($_.Name): $($_.Value)"
    }
    Write-Host ""
}
```

### 3. 检查设备属性名称

WIA 设备属性可能使用不同的名称，常见的有：

- `Name` / `FriendlyName`
- `Manufacturer` / `Mfg`
- `DeviceID`
- `Port`

### 4. 测试扫描功能

创建一个简单的 VBScript 测试：

```vbs
' test_wia.vbs
Set deviceManager = CreateObject("WIA.DeviceManager")
Set deviceInfos = deviceManager.DeviceInfos

WScript.Echo "Found " & deviceInfos.Count & " devices"

For Each deviceInfo In deviceInfos
    If deviceInfo.Type = 1 Then ' Scanner
        WScript.Echo "Scanner: " & deviceInfo.DeviceID

        ' 尝试连接
        On Error Resume Next
        Set device = deviceInfo.Connect()
        If Err.Number = 0 Then
            WScript.Echo "Connected successfully!"

            ' 获取第一个扫描项
            Set item = device.Items(1)
            WScript.Echo "Scanner item available"

            ' 尝试扫描
            Set image = item.Transfer()
            If Err.Number = 0 Then
                WScript.Echo "Scan successful!"
                image.SaveFile "C:\test_scan.jpg"
            Else
                WScript.Echo "Scan failed: " & Err.Description
            End If
        Else
            WScript.Echo "Connect failed: " & Err.Description
        End If
        On Error GoTo 0
    End If
Next
```

运行：
```cmd
cscript test_wia.vbs
```

## 常见错误及解决方案

### 错误 1: "找不到成员"

**原因**: COM 方法或属性名称不正确

**解决方案**:
1. 使用 VBScript 测试正确的方法调用
2. 检查 WIA 版本（WIA 1.0 vs WIA 2.0）
3. 尝试使用 `WIA.CommonDialog` 替代方案

### 错误 2: "访问被拒绝"

**原因**: 权限不足或设备被占用

**解决方案**:
1. 以管理员身份运行 scanserver
2. 关闭其他可能使用扫描仪的程序
3. 重启 Windows Image Acquisition 服务

### 错误 3: "设备不可用"

**原因**: 扫描仪未开机或连接问题

**解决方案**:
1. 检查扫描仪电源
2. 重新插拔 USB 连接
3. 重启扫描仪

## 替代方案：使用 WIA.CommonDialog

如果当前实现有问题，可以尝试使用 CommonDialog：

```go
// 使用 CommonDialog 让用户选择扫描仪
commonDialogRaw, err := oleutil.CreateObject("WIA.CommonDialog")
if err != nil {
    return nil, err
}
commonDialog := commonDialogRaw.ToIDispatch()
defer commonDialog.Release()

// 显示设备选择对话框
deviceRaw, err := oleutil.CallMethod(commonDialog, "ShowSelectDevice")
if err != nil {
    return nil, err
}
device := deviceRaw.ToIDispatch()
defer device.Release()

// 显示扫描对话框
imageRaw, err := oleutil.CallMethod(commonDialog, "ShowAcquireImage")
if err != nil {
    return nil, err
}
image := imageRaw.ToIDispatch()
defer image.Release()
```

## WIA 属性 ID 参考

常用的 WIA 属性 ID：

```
// Device Properties
4098 (0x1002) - WIA_DIP_DEV_NAME
4099 (0x1003) - WIA_DIP_DEV_TYPE
4100 (0x1004) - WIA_DIP_PORT_NAME
4101 (0x1005) - WIA_DIP_DEV_ID
4102 (0x1006) - WIA_DIP_VEND_DESC (Manufacturer)
4103 (0x1007) - WIA_DIP_DEV_DESC (Model)

// Scanner Properties
6146 (0x1802) - WIA_IPS_CUR_INTENT
6147 (0x1803) - WIA_IPS_XRES (Horizontal Resolution)
6148 (0x1804) - WIA_IPS_YRES (Vertical Resolution)
```

## 获取扫描仪信息的改进方法

```go
// 使用属性 ID 而不是名称
func getDeviceProperty(deviceInfo *ole.IDispatch, propID int) (string, error) {
    propsRaw, err := oleutil.GetProperty(deviceInfo, "Properties")
    if err != nil {
        return "", err
    }
    props := propsRaw.ToIDispatch()
    defer props.Release()

    propRaw, err := oleutil.GetProperty(props, "Item", propID)
    if err != nil {
        return "", err
    }
    prop := propRaw.ToIDispatch()
    defer prop.Release()

    valueRaw, err := oleutil.GetProperty(prop, "Value")
    if err != nil {
        return "", err
    }

    return valueRaw.ToString(), nil
}

// 使用
name, _ := getDeviceProperty(deviceInfo, 4098) // WIA_DIP_DEV_NAME
manufacturer, _ := getDeviceProperty(deviceInfo, 4102) // WIA_DIP_VEND_DESC
```

## 调试日志

在代码中添加详细日志：

```go
import "log"

// 在 Scan 方法中
log.Printf("Connecting to scanner: %s", scannerID)
log.Printf("DeviceInfos count: %d", count)
log.Printf("Found device: %s", deviceID)
log.Printf("Calling Connect method...")

deviceRaw, err := oleutil.CallMethod(deviceInfo, "Connect")
if err != nil {
    log.Printf("Connect error: %v", err)
    return nil, fmt.Errorf("failed to connect to scanner: %w", err)
}
log.Printf("Connected successfully")
```

## 下一步操作

1. **测试 PowerShell 脚本** - 确认设备可以被 WIA 访问
2. **运行 VBScript 测试** - 验证完整的扫描流程
3. **更新 scanserver** - 使用新构建的版本
4. **查看详细日志** - 检查具体的错误信息
5. **尝试替代方案** - 如果仍有问题，使用 CommonDialog 或 TWAIN

## 联系支持

如果问题持续，请提供：

1. PowerShell 脚本输出
2. VBScript 测试结果
3. scanserver 完整日志
4. 扫描仪型号和驱动版本
5. Windows 版本信息

这将帮助进一步诊断和解决问题。
