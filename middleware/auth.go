package middleware

import (
	"context"
	"net/http"
	"strings"

	"go-agent-manager/keycloak"

	"github.com/labstack/echo/v4"
)

// 定义一个上下文键，用于存储用户信息
type ContextKey string

const (
	UserKeycloakID ContextKey = "keycloakUserID"
	UserRoles      ContextKey = "keycloakUserRoles"
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
		userID, roles, err := keycloak.ValidateAccessToken(c.Request().Context(), tokenString)
		if err != nil {
			// 根据错误类型返回不同的状态码
			if strings.Contains(err.Error(), "Token is not active") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Token expired or invalid")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Token validation failed: "+err.Error())
		}

		// 将用户信息存储在 Echo 上下文中
		c.Set(string(UserKeycloakID), userID)
		c.Set(string(UserRoles), roles)

		return next(c)
	}
}

// RBACMiddleware 检查用户角色
func RBACMiddleware(requiredRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRoles, ok := c.Get(string(UserRoles)).([]string)
			if !ok {
				return echo.NewHTTPError(http.StatusInternalServerError, "User roles not found in context")
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
