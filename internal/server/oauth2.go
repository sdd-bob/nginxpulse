package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/likaia/nginxpulse/internal/config"
	"golang.org/x/oauth2"
)

// OAuth2 提供商配置
var (
	oauth2Configs   = make(map[string]*oauth2.Config)
	oauth2States    = make(map[string]time.Time) // state -> 过期时间
	oauth2StatesMux sync.RWMutex
	jwtSecretKey    []byte
)

// UserInfo OAuth2 用户信息
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Provider string `json:"provider"`
}

// 初始化 OAuth2 配置
func setupOAuth2(cfg *config.OAuth2Config) {
	if cfg == nil || !cfg.Enabled {
		return
	}

	// 生成 JWT 密钥（生产环境应该从配置文件或环境变量读取固定密钥）
	if jwtSecretKey == nil {
		jwtSecretKey = []byte("nginxpulse-oauth2-jwt-secret-" + time.Now().Format("20060102"))
	}

	// 根据提供商名称自动配置端点
	endpoint := getOAuth2Endpoint(cfg.ProviderName, cfg.AuthURL, cfg.TokenURL)

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = getDefaultScopes(cfg.ProviderName)
	}

	oauth2Configs[cfg.ProviderName] = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}
}

// 获取 OAuth2 端点配置
func getOAuth2Endpoint(providerName, authURL, tokenURL string) oauth2.Endpoint {
	switch providerName {
	case "github":
		return oauth2.Endpoint{
			AuthURL:   "https://github.com/login/oauth/authorize",
			TokenURL:  "https://github.com/login/oauth/access_token",
			AuthStyle: oauth2.AuthStyleInHeader,
		}
	case "google":
		return oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: "https://oauth2.googleapis.com/token",
		}
	default:
		// 自定义提供商
		return oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		}
	}
}

// 获取默认 scopes
func getDefaultScopes(providerName string) []string {
	switch providerName {
	case "github":
		return []string{"read:user", "user:email"}
	case "google":
		return []string{"openid", "email", "profile"}
	default:
		return []string{}
	}
}

// 生成随机 state 防止 CSRF
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// 存储 state（10 分钟过期）
func storeState(state string) {
	oauth2StatesMux.Lock()
	defer oauth2StatesMux.Unlock()
	oauth2States[state] = time.Now().Add(10 * time.Minute)
}

// 验证 state
func validateState(state string) bool {
	oauth2StatesMux.Lock()
	defer oauth2StatesMux.Unlock()

	expireTime, exists := oauth2States[state]
	if !exists || time.Now().After(expireTime) {
		return false
	}

	delete(oauth2States, state)
	return true
}

// 生成 JWT Token
func generateJWTToken(userInfo *UserInfo) (string, error) {
	claims := jwt.MapClaims{
		"user_id":   userInfo.ID,
		"email":     userInfo.Email,
		"name":      userInfo.Name,
		"provider":  userInfo.Provider,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
		"issued_at": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecretKey)
}

// 验证 JWT Token
func verifyJWTToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})
}

// OAuth2 中间件
func oauth2Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过不需要认证的路径
		if skipOAuth2Auth(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 检查 JWT Cookie
		cookie, err := c.Cookie("nginxpulse_jwt")
		if err != nil {
			redirectToLogin(c)
			return
		}

		// 验证 JWT
		token, err := verifyJWTToken(cookie)
		if err != nil || !token.Valid {
			redirectToLogin(c)
			return
		}

		// 将用户信息存入上下文
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			redirectToLogin(c)
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("user_email", claims["email"])
		c.Set("user_name", claims["name"])
		c.Next()
	}
}

// 跳过 OAuth2 认证的路径
func skipOAuth2Auth(path string) bool {
	skipPaths := []string{
		"/auth/",
		"/healthz",
		"/api/status",
	}

	for _, skip := range skipPaths {
		if path == skip || (path != "/api/status" && skip != "/healthz" && skip != "/auth/" && len(path) > len(skip) && path[:len(skip)] == skip) {
			continue
		}
		if path == skip || (len(path) > len(skip) && path[:len(skip)] == skip) {
			return true
		}
	}

	// 静态资源
	if path == "/" || path == "" {
		return true
	}

	return false
}

// 重定向到登录页面
func redirectToLogin(c *gin.Context) {
	// API 请求返回 401，页面请求重定向
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" ||
		c.GetHeader("Accept") == "application/json" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":     "未授权访问",
			"login_url": "/auth/login",
		})
		c.Abort()
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/auth/login")
	c.Abort()
}

// OAuth2 登录处理器
func handleOAuth2Login(c *gin.Context) {
	cfg := config.ReadConfig()
	if cfg.System.OAuth2 == nil || !cfg.System.OAuth2.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "OAuth2 未启用",
		})
		return
	}

	provider := c.DefaultQuery("provider", cfg.System.OAuth2.ProviderName)
	oauthConfig, exists := oauth2Configs[provider]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不支持的 OAuth2 提供商：" + provider,
		})
		return
	}

	// 生成并存储 state
	state, err := generateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成 state 失败",
		})
		return
	}
	storeState(state)

	// 重定向到 OAuth2 提供商
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// OAuth2 回调处理器
func handleOAuth2Callback(c *gin.Context) {
	cfg := config.ReadConfig()
	if cfg.System.OAuth2 == nil || !cfg.System.OAuth2.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "OAuth2 未启用",
		})
		return
	}

	code := c.Query("code")
	state := c.Query("state")
	provider := c.DefaultQuery("provider", cfg.System.OAuth2.ProviderName)

	// 验证 state
	if !validateState(state) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的 state 参数",
		})
		return
	}

	oauthConfig, exists := oauth2Configs[provider]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不支持的 OAuth2 提供商：" + provider,
		})
		return
	}

	// 用 code 换取 token
	ctx := context.Background()
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "授权失败：" + err.Error(),
		})
		return
	}

	// 获取用户信息
	userInfo, err := fetchUserInfo(ctx, provider, token.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取用户信息失败：" + err.Error(),
		})
		return
	}

	userInfo.Provider = provider

	// 生成 JWT Cookie
	jwtToken, err := generateJWTToken(userInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "生成 token 失败",
		})
		return
	}

	// 设置 Cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "nginxpulse_jwt",
		Value:    jwtToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // 生产环境应设为 true（需要 HTTPS）
		MaxAge:   86400, // 24 小时
	})

	// 重定向回首页
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

// 获取用户信息（根据不同提供商实现）
func fetchUserInfo(ctx context.Context, provider, accessToken string) (*UserInfo, error) {
	switch provider {
	case "github":
		return fetchGitHubUserInfo(ctx, accessToken)
	case "google":
		return fetchGoogleUserInfo(ctx, accessToken)
	default:
		// 尝试从标准 OAuth2 userinfo endpoint 获取
		cfg := config.ReadConfig()
		if cfg.System.OAuth2 != nil && cfg.System.OAuth2.UserInfoURL != "" {
			return fetchGenericUserInfo(ctx, cfg.System.OAuth2.UserInfoURL, accessToken)
		}
		return nil, fmt.Errorf("不支持的提供商：%s", provider)
	}
}

// GitHub 用户信息
func fetchGitHubUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "NginxPulse-OAuth2")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 如果 email 为空，尝试从 emails 接口获取
	email := result.Email
	if email == "" {
		email, _ = fetchGitHubPrimaryEmail(ctx, accessToken)
	}

	return &UserInfo{
		ID:       fmt.Sprintf("github:%d", result.ID),
		Email:    email,
		Name:     result.Name,
		Avatar:   result.AvatarURL,
		Provider: "github",
	}, nil
}

// 获取 GitHub 主邮箱
func fetchGitHubPrimaryEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", "NginxPulse-OAuth2")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("no email found")
}

// Google 用户信息
func fetchGoogleUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		Picture   string `json:"picture"`
		GivenName string `json:"given_name"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &UserInfo{
		ID:       fmt.Sprintf("google:%s", result.ID),
		Email:    result.Email,
		Name:     result.Name,
		Avatar:   result.Picture,
		Provider: "google",
	}, nil
}

// 通用用户信息获取
func fetchGenericUserInfo(ctx context.Context, userInfoURL, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 尝试提取常见字段
	getString := func(key string) string {
		if v, ok := result[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	// 获取 avatar，尝试多个可能的字段名
	avatar := getString("avatar")
	if avatar == "" {
		avatar = getString("picture")
	}
	if avatar == "" {
		avatar = getString("avatar_url")
	}

	return &UserInfo{
		ID:       getString("id"),
		Email:    getString("email"),
		Name:     getString("name"),
		Avatar:   avatar,
		Provider: "custom",
	}, nil
}

// 登出处理器
func handleLogout(c *gin.Context) {
	// 清除 JWT Cookie
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "nginxpulse_jwt",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "已退出登录",
	})
}

// 认证状态查询
func handleAuthStatus(c *gin.Context) {
	cookie, err := c.Cookie("nginxpulse_jwt")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"logged_in": false,
		})
		return
	}

	token, err := verifyJWTToken(cookie)
	if err != nil || !token.Valid {
		c.JSON(http.StatusOK, gin.H{
			"logged_in": false,
		})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"logged_in": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logged_in": true,
		"user_id":   claims["user_id"],
		"email":     claims["email"],
		"name":      claims["name"],
		"provider":  claims["provider"],
	})
}
