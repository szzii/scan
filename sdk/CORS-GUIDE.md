# CORS 跨域问题解决方案

## 什么是CORS？

CORS (Cross-Origin Resource Sharing，跨域资源共享) 是浏览器的一种安全机制。当你的网页从一个域名访问另一个域名的API时，浏览器会阻止这个请求。

例如：
- 你的网页: `http://localhost:3000`
- 扫描服务: `http://localhost:8080`

这两个端口不同，浏览器认为是不同的域，会触发CORS限制。

## ✅ 已解决

**扫描服务已内置CORS支持！**

从 v1.0.13+ 版本开始，服务器已经配置了CORS中间件，允许所有域名跨域访问。

### CORS配置详情

服务器已设置以下响应头：

```
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
Access-Control-Allow-Headers: Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With
Access-Control-Allow-Methods: POST, OPTIONS, GET, PUT, DELETE
```

**这意味着：**
- ✅ 任何域名都可以访问API
- ✅ 支持所有常用HTTP方法
- ✅ 支持所有常用请求头
- ✅ 自动处理OPTIONS预检请求

## 使用示例

### 从不同端口访问

```javascript
// 你的React/Vue应用运行在 http://localhost:3000
// 扫描服务运行在 http://localhost:8080

const client = new ScannerClient('http://localhost:8080');

// 这个请求会成功，不会有CORS错误
const scanners = await client.listScanners();
```

### 从不同域名访问

```javascript
// 你的网站: https://myapp.com
// 扫描服务: http://192.168.1.100:8080

const client = new ScannerClient('http://192.168.1.100:8080');

// 也可以正常工作
const scanners = await client.listScanners();
```

## 常见问题

### Q1: 我仍然看到CORS错误

**可能原因：**

1. **使用的是旧版本服务器**
   - 解决方案：升级到 v1.0.13+ 版本

2. **服务器未启动**
   - 解决方案：确保扫描服务正在运行
   - 测试：在浏览器访问 `http://localhost:8080/api/v1/health`

3. **URL错误**
   - 解决方案：检查 `ScannerClient` 的 baseUrl 是否正确
   - 示例：`new ScannerClient('http://localhost:8080')` （不要加 /api/v1）

4. **浏览器缓存**
   - 解决方案：清除浏览器缓存或使用无痕模式

### Q2: 如何限制允许的域名？

默认配置允许所有域名访问（`Access-Control-Allow-Origin: *`）。

如果你需要限制特定域名，可以修改服务器代码：

**修改文件:** `internal/api/server.go`

```go
// 将这行：
c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

// 改为指定域名：
c.Writer.Header().Set("Access-Control-Allow-Origin", "https://myapp.com")

// 或根据请求动态设置：
origin := c.Request.Header.Get("Origin")
if origin == "https://myapp.com" || origin == "http://localhost:3000" {
    c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
}
```

### Q3: WebSocket也有CORS问题吗？

WebSocket不受CORS限制！但要注意：

1. **协议匹配**
   - HTTP网站 → `ws://` WebSocket
   - HTTPS网站 → `wss://` WebSocket

2. **混合内容**
   - HTTPS网站不能连接 `ws://`（非加密）WebSocket
   - 只能连接 `wss://`（加密）WebSocket

**示例：**

```javascript
// HTTP网站
const client = new ScannerClient('http://localhost:8080');  // ✅ OK

// HTTPS网站
const client = new ScannerClient('https://localhost:8080'); // ✅ OK
const client = new ScannerClient('http://localhost:8080');  // ❌ Mixed content error
```

## 开发环境配置

### React开发服务器

如果你使用 `create-react-app`，也可以配置代理来避免CORS：

**package.json:**
```json
{
  "proxy": "http://localhost:8080"
}
```

**使用代理后：**
```javascript
// 直接使用相对路径
const client = new ScannerClient('');  // 自动代理到 localhost:8080
```

### Vue.js开发服务器

**vue.config.js:**
```javascript
module.exports = {
  devServer: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/ws': {
        target: 'http://localhost:8080',
        ws: true
      }
    }
  }
}
```

### Vite开发服务器

**vite.config.js:**
```javascript
export default {
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true
      }
    }
  }
}
```

## 生产环境

### 方案1: 使用反向代理（推荐）

使用Nginx作为反向代理，统一域名：

```nginx
server {
    listen 80;
    server_name myapp.com;

    # 前端应用
    location / {
        proxy_pass http://localhost:3000;
    }

    # 扫描API
    location /api/ {
        proxy_pass http://localhost:8080;
    }

    # WebSocket
    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

**使用反向代理后：**
```javascript
// 前端和API使用同一域名，不需要跨域
const client = new ScannerClient('');  // 使用当前域名
```

### 方案2: 配置HTTPS

如果扫描服务需要HTTPS：

1. 生成SSL证书
2. 配置服务器支持HTTPS
3. 使用 `https://` 协议

```javascript
const client = new ScannerClient('https://scanner.myapp.com');
```

## 测试CORS配置

### 使用curl测试

```bash
# 测试API访问
curl -i -X GET http://localhost:8080/api/v1/scanners \
  -H "Origin: http://localhost:3000"

# 应该看到响应头：
# Access-Control-Allow-Origin: *
```

### 使用浏览器测试

```javascript
// 打开浏览器控制台
fetch('http://localhost:8080/api/v1/health')
  .then(r => r.json())
  .then(console.log)
  .catch(console.error)

// 如果成功，说明CORS配置正确
```

### 使用SDK测试

```javascript
const client = new ScannerClient('http://localhost:8080');

client.healthCheck()
  .then(result => {
    console.log('✅ CORS配置正确！');
    console.log('Health check:', result);
  })
  .catch(error => {
    console.error('❌ CORS错误:', error);
  });
```

## 安全建议

### 开发环境
- ✅ 使用 `Access-Control-Allow-Origin: *` （方便开发）

### 生产环境
- ✅ 限制允许的域名
- ✅ 使用反向代理统一域名
- ✅ 启用HTTPS
- ✅ 配置防火墙规则

### 内网环境
- ✅ 如果只在内网使用，可以保持 `*` 配置
- ✅ 通过网络隔离保证安全

## 故障排查

### 步骤1: 检查服务器版本
```bash
# 查看CHANGELOG确认版本
cat CHANGELOG.md | head -20
```

### 步骤2: 检查服务器响应头
打开浏览器开发者工具 → Network → 查看响应头

应该看到：
```
Access-Control-Allow-Origin: *
```

### 步骤3: 检查请求方法
确保使用的是允许的HTTP方法：
- GET, POST, PUT, DELETE, OPTIONS

### 步骤4: 检查请求头
确保使用的是允许的请求头：
- Content-Type, Authorization, etc.

### 步骤5: 查看控制台错误
完整的错误信息会显示具体原因：
```
Access to fetch at 'http://localhost:8080/api/v1/scanners' from origin 'http://localhost:3000' has been blocked by CORS policy
```

## 总结

✅ **扫描服务已内置CORS支持**
✅ **默认允许所有域名访问**
✅ **自动处理预检请求**
✅ **无需额外配置即可使用**

如果仍有问题，请检查：
1. 服务器版本（需要 v1.0.13+）
2. 服务器是否正在运行
3. URL是否正确
4. 浏览器缓存

---

**需要帮助？**

查看完整文档：
- API文档: `SDK-README.md`
- 快速开始: `README.md`
- 示例代码: `example.html`
