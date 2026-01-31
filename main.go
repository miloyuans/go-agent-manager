package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"go-agent-manager/config"
	"go-agent-manager/db"
	"go-agent-manager/handlers"
	"go-agent-manager/keycloak"
	"go-agent-manager/middleware"

	"github.com/labstack/echo/v4"
	e_middleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. 加载配置
	config.LoadConfig()

	// 2. 初始化数据库
	db.InitDB()

	// 3. 初始化 Keycloak 客户端
	keycloak.InitKeycloak()

	// 4. 创建 Echo 实例
	e := echo.New()

	// 5. 注册全局中间件
	e.Use(e_middleware.Logger()) // 请求日志
	e.Use(e_middleware.Recover()) // 崩溃恢复
	e.Use(middleware.CORSMiddleware()) // CORS 允许跨域

	// 6. 静态文件服务 (前端构建后的 dist 目录)
	// 在生产环境中，Go 后端会托管前端静态文件
	// 在开发环境中，前端由 Vite Dev Server 提供，这里可以暂时不启用或只用于生产构建
	frontendPath := config.AppConfig.FrontendStaticPath
	if _, err := os.Stat(frontendPath); err == nil {
		// 路由任何不匹配 API 的请求都由 ServeFrontend 处理
		e.GET("/*", handlers.ServeFrontend())
		log.Printf("Frontend static file serving enabled from: %s", frontendPath)
	} else {
		log.Printf("Frontend static path %s not found or inaccessible. Static file serving disabled. Error: %v", frontendPath, err)
	}


	// 7. API 路由组
	apiGroup := e.Group("/api")

	// 注册 Keycloak 认证中间件到 API 路由组
	apiGroup.Use(middleware.KeycloakAuthMiddleware)

	// 定义需要管理员角色的路由
	adminGroup := apiGroup.Group("/admin")
	adminGroup.Use(middleware.RBACMiddleware("admin")) // 只有 'admin' 角色能访问

	// --- 设备管理 (需要管理员角色) ---
	adminGroup.GET("/devices", handlers.GetDevices)
	adminGroup.POST("/devices", handlers.CreateDevice) // Agent 上报设备也可以走这里，如果 Agent 也是用户身份
	adminGroup.PUT("/devices/:id", handlers.UpdateDevice)
	adminGroup.DELETE("/devices/:id", handlers.DeleteDevice)

	// --- 用户管理 (需要管理员角色) ---
	adminGroup.GET("/users", handlers.GetUsers) // 从 Keycloak 获取用户列表
	adminGroup.PUT("/users/:id/status", handlers.UpdateUserStatus) // 启用/禁用 Keycloak 用户

	// --- 绑定管理 (需要管理员角色) ---
	adminGroup.GET("/bindings", handlers.GetBindings)
	adminGroup.POST("/bindings", handlers.CreateBinding)
	adminGroup.DELETE("/bindings/:id", handlers.DeleteBinding)

	// --- 规则管理 (需要管理员角色) ---
	adminGroup.GET("/rules", handlers.GetRules)
	adminGroup.POST("/rules", handlers.CreateRule)
	adminGroup.PUT("/rules/:id", handlers.UpdateRule)
	adminGroup.DELETE("/rules/:id", handlers.DeleteRule)

	// 8. 启动服务器
	log.Printf("Server starting on port %s", config.AppConfig.ServerPort)
	if err := e.Start(":" + config.AppConfig.ServerPort); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server stopped with error: %v", err)
	}
}
