package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"go-agent-manager/config"

	"github.com/labstack/echo/v4"
)

// ServeFrontend 托管前端静态文件
func ServeFrontend() echo.HandlerFunc {
	staticPath := config.AppConfig.FrontendStaticPath
	log.Printf("Serving frontend static files from: %s", staticPath)

	return func(c echo.Context) error {
		reqPath := c.Request().URL.Path

		// 构建文件系统路径
		fsPath := filepath.Join(staticPath, reqPath)

		// 检查文件是否存在
		_, err := os.Stat(fsPath)
		if err == nil { // 文件存在，直接提供服务
			return c.File(fsPath)
		}

		// 如果文件不存在，但请求的不是根路径，则尝试提供 index.html (适用于 SPA 的 History 模式)
		if os.IsNotExist(err) && !isFilePath(reqPath) {
			indexPath := filepath.Join(staticPath, "index.html")
			return c.File(indexPath)
		}

		// 其他情况，文件不存在，返回 404
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}
}

// isFilePath 辅助函数，检查路径是否看起来像一个文件路径 (包含扩展名)
func isFilePath(path string) bool {
	return len(filepath.Ext(path)) > 0
}
