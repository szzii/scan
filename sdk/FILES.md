# Scanner SDK 文件说明

## 📦 scanner-sdk-v1.0.0.zip (21 KB)

完整的JavaScript SDK包，包含以下7个文件：

### 核心文件

#### 1. scanner-sdk.js (12 KB)
**SDK主文件** - 可直接在浏览器或Node.js中使用

**使用方法：**
```html
<script src="scanner-sdk.js"></script>
<script>
    const client = new ScannerClient('http://localhost:8080');
</script>
```

**功能：**
- 完整的API封装
- WebSocket实时更新
- Promise-based异步调用
- 自动错误处理
- 零依赖

---

### 文档文件

#### 2. README.md (7 KB)
**快速开始指南**

**包含内容：**
- 快速开始示例
- API方法列表
- React/Vue集成示例
- 使用场景说明
- CORS支持说明

**适合：** 快速上手，了解基本用法

---

#### 3. SDK-README.md (13 KB)
**完整API文档**

**包含内容：**
- 详细的API参数说明
- 所有方法的完整示例
- 事件监听详解
- React/Vue/Node.js集成示例
- 错误处理指南
- 浏览器兼容性

**适合：** 深入学习，查阅API细节

---

#### 4. CORS-GUIDE.md (8 KB)
**CORS跨域完整指南**

**包含内容：**
- CORS概念详解
- 常见问题Q&A（10+个问题）
- 开发环境配置（React/Vue/Vite代理）
- 生产环境部署（Nginx反向代理）
- 故障排查步骤
- 安全建议

**适合：** 解决跨域问题，生产环境部署

---

### 演示文件

#### 5. example.html (16 KB)
**交互式演示页面**

**功能：**
- 扫描仪列表加载
- 参数配置界面
- 实时进度显示
- 扫描结果预览
- 操作日志监控

**使用：**
1. 修改页面中的 `SERVER_URL`
2. 用浏览器打开 example.html
3. 点击"刷新扫描仪列表"
4. 选择扫描仪并配置参数
5. 开始扫描

**适合：** 快速体验SDK功能，学习使用方法

---

#### 6. test-cors.html (9 KB)
**CORS测试工具**

**功能：**
- 自动测试CORS配置
- 检查API连接
- 显示详细测试结果
- 提供故障排查建议

**使用：**
1. 用浏览器打开 test-cors.html
2. 输入扫描服务地址
3. 点击"测试CORS"
4. 查看测试结果

**适合：** 诊断跨域问题，验证CORS配置

---

### 配置文件

#### 7. package.json (398 B)
**NPM包配置**

**内容：**
```json
{
  "name": "scanner-service-sdk",
  "version": "1.0.0",
  "description": "JavaScript SDK for Scanner Service API",
  "main": "scanner-sdk.js",
  "keywords": ["scanner", "scan", "wia", "twain", "sdk"]
}
```

**适合：** NPM发布，项目依赖管理

---

## 🚀 快速开始

### 1. 解压SDK包
```bash
unzip scanner-sdk-v1.0.0.zip
```

### 2. 选择使用方式

#### 方式A: 浏览器直接使用
```html
<script src="scanner-sdk.js"></script>
<script>
    const client = new ScannerClient('http://localhost:8080');

    // 获取扫描仪
    client.listScanners().then(scanners => {
        console.log(scanners);
    });
</script>
```

#### 方式B: ES6模块
```javascript
import ScannerClient from './scanner-sdk.js';

const client = new ScannerClient('http://localhost:8080');
const scanners = await client.listScanners();
```

#### 方式C: Node.js
```javascript
const ScannerClient = require('./scanner-sdk.js');

const client = new ScannerClient('http://localhost:8080');
// 使用API...
```

### 3. 查看演示
```bash
# 用浏览器打开
open example.html
```

### 4. 测试CORS
```bash
# 用浏览器打开
open test-cors.html
```

---

## 📖 文档导航

| 需求 | 查看文档 |
|------|----------|
| 快速上手 | README.md |
| API详细说明 | SDK-README.md |
| 解决跨域问题 | CORS-GUIDE.md |
| 查看示例代码 | example.html |
| 测试CORS | test-cors.html |

---

## 🌟 功能特性

### SDK功能
- ✅ 获取扫描仪列表
- ✅ 创建扫描任务
- ✅ 批量扫描
- ✅ 任务管理（列表/查询/取消）
- ✅ WebSocket实时更新
- ✅ 进度监控
- ✅ 文件URL生成
- ✅ 健康检查

### 跨域支持
- ✅ 内置CORS支持（v1.0.14+）
- ✅ 允许所有域名访问
- ✅ 自动处理预检请求
- ✅ 无需额外配置

### 框架集成
- ✅ 原生JavaScript
- ✅ React
- ✅ Vue.js
- ✅ Angular
- ✅ Node.js

---

## 📦 文件大小

| 文件 | 大小 | 说明 |
|------|------|------|
| scanner-sdk.js | 12 KB | 核心SDK |
| SDK-README.md | 13 KB | API文档 |
| example.html | 16 KB | 演示页面 |
| test-cors.html | 9 KB | 测试工具 |
| CORS-GUIDE.md | 8 KB | CORS指南 |
| README.md | 7 KB | 快速开始 |
| package.json | 398 B | NPM配置 |
| **总计** | **21 KB** | **完整SDK包** |

---

## 🔄 版本信息

- **当前版本：** v1.0.0
- **发布日期：** 2025-11-10
- **兼容性：** 需要扫描服务 v1.0.14+（支持CORS）

---

## 📝 更新日志

### v1.0.0 (2025-11-10)
- ✅ 初始发布
- ✅ 完整的API封装
- ✅ WebSocket实时更新
- ✅ CORS跨域支持
- ✅ 完整文档和示例
- ✅ CORS测试工具

---

## 🤝 技术支持

### 文档
- 快速开始：README.md
- API文档：SDK-README.md
- CORS指南：CORS-GUIDE.md

### 示例
- 演示页面：example.html
- 测试工具：test-cors.html

### 问题排查
1. 查看 CORS-GUIDE.md 的常见问题章节
2. 使用 test-cors.html 测试连接
3. 查看浏览器控制台错误信息
4. 确认服务器版本（需要 v1.0.14+）

---

## 📄 许可证

MIT License

---

**快速链接：**
- [快速开始](./README.md)
- [完整API文档](./SDK-README.md)
- [CORS指南](./CORS-GUIDE.md)
- [演示页面](./example.html)
- [测试工具](./test-cors.html)
