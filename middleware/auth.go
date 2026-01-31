package middleware

import (
	"net/http"
	"strings"

	"go-agent-manager/keycloak"

	"github.com/labstack/echo/v4"
)

// 定义一个上下文键，用于存储用户信息
// 尽管 keycloak.ValidateAccessToken 需要 context，但在这个中间件里
// 我们主要使用 echo.Context 的 Request().Context()，不需要显式创建新 context
// 所以移除了 "context" 包的导入

const (
	UserKeycloakID = "keycloakUserID"
	UserRoles      = "keycloakUserRoles"
)

// KeycloakAuthMiddleware 验证 JWT 并将用户信息添加到上下文中
func KeycloakAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "Authorization header is required")
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid Authorization header format")
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// 使用 Keycloak 模块验证 token
		// c.Request().Context() 是 http.Request 的上下文，会被 ValidateAccessToken 使用
		userID, roles, err := keycloak.ValidateAccessToken(c.Request().Context(), tokenString)
		if err != nil {
			// 根据错误类型返回不同的状态码
			if strings.Contains(err.Error(), "token is not active") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Token expired or invalid")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Token validation failed: "+err.Error())
		}

		// 将用户信息存储在 Echo 上下文中
		c.Set(UserKeycloakID, userID)
		c.Set(UserRoles, roles)

		return next(c)
	}
}

// RBACMiddleware 检查用户角色
func RBACMiddleware(requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRoles, ok := c.Get(UserRoles).([]string)
			if !ok {
				// 如果之前的 Auth 中间件成功了，这里理论上不应该发生，除非是逻辑错误
				return echo.NewHTTPError(http.StatusForbidden, "User roles not found context")
			}

			// 检查用户是否拥有所需的所有角色中的至少一个
			hasRole := false
			for _, requiredRole := range requiredRoles {
				for _, userRole := range userRoles {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break // 找到了一个匹配的角色，跳出循环
				}
			}

			if !hasRole {
				return echo.NewHTTPError(http.StatusForbidden, "Forbidden: insufficient roles")
			}
			return next(c)
		}
	}
}
