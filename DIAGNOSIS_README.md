# 扫描仪检测诊断工具

## 问题
D2800+ 扫描仪无法检测到

## 诊断步骤

### 1. 运行诊断工具

在 Windows 上，导航到 `build` 目录，运行：

```
run_diagnose.bat
```

或者直接运行：

```
diagnose.exe
```

### 2. 查看输出

诊断工具会测试：

**TEST 1: WIA Scanner Detection**
- 检测 WIA 2.0 (Vista+) 或 WIA 1.0 (XP)
- 列出所有 WIA 设备
- 显示设备名称、ID、类型

**TEST 2: TWAIN Scanner Detection**
- 检测 TWAINDSM.dll 是否安装
- 打开 TWAIN Data Source Manager
- 枚举所有 TWAIN 数据源
- 显示产品名称、制造商、系列

### 3. 期望结果

**如果 D2800+ 是 WIA 扫描仪：**
```
TEST 1: WIA Scanner Detection
------------------------------
✓ WIA 2.0 available
Found 1 WIA device(s)

  Device 1:
    Name: HP ScanJet Pro 2000 s2 D2800+
    ID: {xxxxx}
    Type: 1 (Scanner)
```

**如果 D2800+ 是 TWAIN 扫描仪：**
```
TEST 2: TWAIN Scanner Detection
--------------------------------
✓ TWAIN DSM found (TWAINDSM.dll)
✓ DSM_Entry function found
✓ DSM opened successfully

  Data Source 1:
    Product: HP ScanJet Pro 2000 s2
    Manufacturer: Hewlett-Packard
    Family: D2800+
    ID: 12345
```

### 4. 可能的问题

#### 问题 1: WIA 不可用
```
❌ WIA 2.0 not available
❌ WIA 1.0 also not available
```
**解决方案：**
- 确保 Windows 映像获取服务 (WIA) 正在运行
- 控制面板 → 服务 → Windows Image Acquisition (WIA)

#### 问题 2: TWAIN DSM 未安装
```
❌ TWAIN DSM not found (TWAINDSM.dll)
```
**解决方案：**
- 下载安装 TWAIN DSM：https://www.twain.org/
- 或者安装扫描仪驱动时选择包含 TWAIN 支持

#### 问题 3: 找不到 TWAIN 数据源
```
✓ TWAIN DSM found
✓ DSM opened successfully
❌ No TWAIN data sources found
```
**解决方案：**
- 重新安装 HP ScanJet D2800+ 驱动程序
- 确保安装时选择了 TWAIN 驱动

#### 问题 4: WIA 和 TWAIN 都找不到扫描仪
**可能原因：**
- 扫描仪未连接
- 驱动程序未正确安装
- USB 端口问题（网络扫描仪则为网络连接问题）
- 扫描仪未开机

**解决方案：**
1. 检查扫描仪连接和电源
2. 重新安装驱动程序
3. 在"设备和打印机"中查看扫描仪是否显示
4. 尝试使用 HP 官方扫描软件测试

### 5. 报告问题

请运行诊断工具并将完整输出复制给我，包括：
- WIA 检测结果
- TWAIN 检测结果
- 任何错误消息

这将帮助我准确定位问题所在。

## 常见情况

### 情况 A: D2800+ 在 WIA 中找到，但主程序中看不到
- **原因**：Combined driver 的 WIA 部分可能有 bug
- **解决**：需要修复 driver_windows.go

### 情况 B: D2800+ 在 TWAIN 中找到，但主程序中看不到
- **原因**：Combined driver 的 TWAIN 部分可能有 bug
- **解决**：需要修复 driver_windows_twain.go

### 情况 C: 诊断工具也找不到 D2800+
- **原因**：驱动程序安装问题
- **解决**：重新安装 HP ScanJet D2800+ 驱动程序

### 情况 D: 诊断工具找到其他扫描仪但没有 D2800+
- **原因**：D2800+ 驱动未安装或连接问题
- **解决**：检查物理连接和驱动安装

## HP ScanJet Pro 2000 s2 D2800+ 信息

**产品页面：**
https://support.hp.com/cn-zh/drivers/hp-scanjet-pro-2000-s2-scanner-series

**支持的协议：**
- WIA 2.0 (推荐)
- TWAIN

**驱动下载：**
请从 HP 官网下载最新驱动

**网络扫描仪配置：**
如果是网络版本，确保：
1. 扫描仪 IP 地址可达
2. Windows 防火墙允许 WIA 服务
3. 网络扫描仪已正确添加到"设备和打印机"

---

运行诊断工具后，请将输出发给我，我会帮助分析问题！
