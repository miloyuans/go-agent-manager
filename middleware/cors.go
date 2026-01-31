package middleware

import (
	"github.com/labstack/echo/v4"
	e_middleware "github.com/labstack/echo/v4/middleware"
)

// CORSMiddleware 配置 CORS
func CORSMiddleware() echo.MiddlewareFunc {
	return e_middleware.CORSWithConfig(e_middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // 生产环境中应限制为前端域名
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	})
}
