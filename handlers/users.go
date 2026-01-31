package handlers

import (
	"context"
	"net/http"
	"go-agent-manager/keycloak" // 引入 keycloak 模块
	"github.com/labstack/echo/v4"
)

// GetUsers 获取 Keycloak 用户列表
func GetUsers(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	users, err := keycloak.FetchKeycloakUsers(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch users from Keycloak: "+err.Error())
	}
	return c.JSON(http.StatusOK, users)
}

// UpdateUserStatus 启用或禁用 Keycloak 用户
func UpdateUserStatus(c echo.Context) error {
	userID := c.Param("id")
	type StatusUpdate struct {
		Enabled bool `json:"enabled"`
	}
	su := new(StatusUpdate)
	if err := c.Bind(su); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	err := keycloak.UpdateKeycloakUserStatus(ctx, userID, su.Enabled)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update user status in Keycloak: "+err.Error())
	}
	return c.NoContent(http.StatusOK)
}
