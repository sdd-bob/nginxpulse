# NginxPulse OAuth2 嵌入支持指南

## 概述

NginxPulse 现已支持 OAuth2 认证，可以实现无感嵌入到第三方平台中。通过 OAuth2，用户可以使用 GitHub、Google 等第三方账号登录，或者对接自定义的 SSO 系统。

## 特性

- **标准 OAuth2 协议**：支持授权码模式，兼容主流身份提供商
- **多提供商支持**：内置 GitHub、Google 配置，支持自定义 OAuth2 提供商
- **分层认证**：OAuth2 用于用户登录，Access Key 保留用于 API/服务间调用
- **JWT Session**：基于 JWT 的会话管理，支持自定义超时时间
- **无感嵌入**：支持 iframe 嵌入，已登录用户无需重复认证

## 快速开始

### 1. 使用环境变量配置（推荐）

```bash
# 启用 OAuth2
export OAUTH2_ENABLED=true

# 选择提供商（github, google, custom）
export OAUTH2_PROVIDER_NAME=github

# 填入你的 OAuth2 凭证
export OAUTH2_CLIENT_ID="your-client-id"
export OAUTH2_CLIENT_SECRET="your-client-secret"

# 设置回调地址（必须与提供商配置一致）
export OAUTH2_REDIRECT_URL="http://localhost:8088/auth/callback"

# 授权范围（逗号分隔）
export OAUTH2_SCOPES="read:user,user:email"

# 启动服务
./nginxpulse
```

### 2. 使用配置文件

创建 `configs/nginxpulse_config.json`：

```json
{
  "system": {
    "oauth2": {
      "enabled": true,
      "providerName": "github",
      "clientID": "your-github-client-id",
      "clientSecret": "your-github-client-secret",
      "redirectURL": "http://localhost:8088/auth/callback",
      "scopes": ["read:user", "user:email"],
      "sessionTimeout": "24h"
    }
  },
  "server": {
    "Port": ":8088"
  },
  "database": {
    "driver": "postgres",
    "dsn": "postgres://user:password@localhost:5432/nginxpulse?sslmode=disable"
  },
  "websites": [...]
}
```

## 配置 OAuth2 提供商

### GitHub

1. 访问 https://github.com/settings/developers
2. 创建一个新的 OAuth App
3. 填写 Authorization callback URL：`http://localhost:8088/auth/callback`
4. 获取 Client ID 和 Client Secret
5. 配置环境变量：

```bash
OAUTH2_PROVIDER_NAME=github
OAUTH2_CLIENT_ID=your_client_id
OAUTH2_CLIENT_SECRET=your_client_secret
OAUTH2_SCOPES=read:user,user:email
```

### Google

1. 访问 https://console.cloud.google.com/apis/credentials
2. 创建新的 OAuth 2.0 Client ID
3. 添加授权重定向 URI：`http://localhost:8088/auth/callback`
4. 获取 Client ID 和 Client Secret
5. 配置环境变量：

```bash
OAUTH2_PROVIDER_NAME=google
OAUTH2_CLIENT_ID=your_client_id
OAUTH2_CLIENT_SECRET=your_client_secret
OAUTH2_SCOPES=openid,email,profile
```

### 自定义 OAuth2 提供商

```bash
OAUTH2_PROVIDER_NAME=custom
OAUTH2_AUTH_URL=https://your-idp.com/oauth/authorize
OAUTH2_TOKEN_URL=https://your-idp.com/oauth/token
OAUTH2_USER_INFO_URL=https://your-idp.com/oauth/userinfo
OAUTH2_CLIENT_ID=your_client_id
OAUTH2_CLIENT_SECRET=your_client_secret
OAUTH2_REDIRECT_URL=http://localhost:8088/auth/callback
```

## 嵌入到第三方平台

### iframe 嵌入方式

```html
<!-- 简单嵌入 -->
<iframe 
  src="http://localhost:8088" 
  width="100%" 
  height="600"
  style="border: none;"
></iframe>

<!-- 隐藏侧边栏嵌入 -->
<iframe 
  src="http://localhost:8088?embed=true" 
  width="100%" 
  height="600"
  style="border: none;"
></iframe>
```

### 微前端集成

在主应用中配置单点登录：

```javascript
// 主应用认证后，用户访问 NginxPulse 时会自动携带 JWT Cookie
// 无需额外配置
```

## 认证流程

```
1. 用户访问 NginxPulse
   ↓
2. 检测未登录，重定向到 OAuth2 提供商
   ↓
3. 用户授权登录
   ↓
4. 回调 /auth/callback 并获取 code
   ↓
5. 用 code 换取 access_token
   ↓
6. 获取用户信息
   ↓
7. 生成 JWT Cookie
   ↓
8. 重定向回首页（已登录状态）
```

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/auth/login` | GET | 发起 OAuth2 授权 |
| `/auth/callback` | GET | OAuth2 回调处理 |
| `/auth/logout` | POST | 退出登录 |
| `/auth/status` | GET | 查询当前登录状态 |

## 安全建议

1. **生产环境必须使用 HTTPS**：Cookie 需要设置 `Secure` 标志
2. **JWT 密钥**：从环境变量读取固定的密钥，不要使用默认值
3. **CORS 配置**：限制具体的 `AllowOrigins`，不要使用 `*`
4. **Session 超时**：根据业务需求合理设置 `OAUTH2_SESSION_TIMEOUT`
5. **Redirect URL 白名单**：严格限制回调地址

## 故障排查

### 问题：登录后仍然显示登录页面

**原因**：JWT Cookie 未正确设置或验证失败

**解决**：
1. 检查浏览器控制台是否有错误
2. 确认 OAuth2 回调地址配置正确
3. 查看服务器日志中的认证信息

### 问题：iframe 中无法登录

**原因**：浏览器的 SameSite Cookie 策略限制

**解决**：
1. 确保使用 HTTPS
2. 在 oauth2.go 中设置 Cookie 的 `SameSite` 为 `None`

```go
http.SetCookie(c.Writer, &http.Cookie{
    Name:     "nginxpulse_jwt",
    Value:    jwtToken,
    Path:     "/",
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteNoneMode, // 允许跨站 Cookie
    MaxAge:   86400,
})
```

### 问题：自定义提供商返回用户信息为空

**原因**：UserInfo URL 返回的字段名不匹配

**解决**：检查自定义提供商返回的 JSON 格式，修改 `fetchGenericUserInfo` 函数中的字段映射。

## 环境变量完整列表

```bash
# OAuth2 基础配置
OAUTH2_ENABLED=false              # 是否启用 OAuth2
OAUTH2_PROVIDER_NAME=github       # 提供商名称
OAUTH2_CLIENT_ID=                 # 客户端 ID
OAUTH2_CLIENT_SECRET=             # 客户端密钥
OAUTH2_REDIRECT_URL=              # 回调地址
OAUTH2_SCOPES=                    # 授权范围（逗号分隔）
OAUTH2_SESSION_TIMEOUT=24h        # Session 超时时间

# 自定义提供商端点（可选）
OAUTH2_AUTH_URL=                  # 授权页面地址
OAUTH2_TOKEN_URL=                 # Token 交换地址
OAUTH2_USER_INFO_URL=             # 用户信息接口地址
```

## 混合认证模式

NginxPulse 支持 OAuth2 和 Access Key 同时存在：

- **OAuth2**：用于浏览器用户的交互式登录
- **Access Key**：用于 API 调用、自动化脚本等服务间通信

启用 OAuth2 后，现有的 Access Key 仍然有效，两者互不影响。

## 示例项目

参考 `configs/oauth2-example.json` 和 `.env.example` 获取完整的配置示例。
