package server

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/likaia/nginxpulse/internal/analytics"
	"github.com/likaia/nginxpulse/internal/config"
	"github.com/likaia/nginxpulse/internal/ingest"
	"github.com/likaia/nginxpulse/internal/web"
	"github.com/sirupsen/logrus"
)

// StartHTTPServer configures and starts the HTTP server in a goroutine.
func StartHTTPServer(statsFactory *analytics.StatsFactory, logParser *ingest.LogParser, addr string) (*http.Server, error) {
	router := buildRouter(statsFactory, logParser)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("HTTP 服务器运行失败")
		}
	}()

	logrus.Infof("服务器已启动，监听地址: %s", addr)
	return server, nil
}

func buildRouter(statsFactory *analytics.StatsFactory, logParser *ingest.LogParser) *gin.Engine {
	cfg := config.ReadConfig()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(requestLogger())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", accessKeyHeader},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	router.Use(basePathMiddleware(router))

	// 认证中间件：OAuth2 优先，否则使用 Access Key
	oauth2Enabled := cfg.System.OAuth2 != nil && cfg.System.OAuth2.Enabled
	if oauth2Enabled {
		setupOAuth2(cfg.System.OAuth2) // 初始化 OAuth2 配置
		router.Use(oauth2Middleware())

		// 注册 OAuth2 路由
		router.GET("/auth/login", handleOAuth2Login)
		router.GET("/auth/callback", handleOAuth2Callback)
		router.POST("/auth/logout", handleLogout)
		router.GET("/auth/status", handleAuthStatus)
	} else {
		router.Use(accessKeyMiddleware())
	}

	router.GET("/healthz", gin.WrapF(HealthHandler))

	web.SetupRoutes(router, statsFactory, logParser)
	attachAppConfig(router)
	attachWebUI(router)

	return router
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()

		if status >= 400 {
			logrus.Warnf("HTTP %d %s %s %s %v",
				status, c.Request.Method, path, c.ClientIP(), duration)
			return
		}

		if strings.HasPrefix(path, "/api/") && duration > 100*time.Millisecond {
			logrus.Warnf("高延迟 %s %s %d %s %v",
				c.Request.Method, path, status, c.ClientIP(), duration)
		}
	}
}
