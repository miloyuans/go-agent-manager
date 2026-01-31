package handlers

import (
	"net/http"

	"go-agent-manager/db"
	"go-agent-manager/models"

	"github.com/labstack/echo/v4"
)

// GetRules 获取所有代理规则
func GetRules(c echo.Context) error {
	var rules []models.Rule
	if result := db.DB.Find(&rules); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.JSON(http.StatusOK, rules)
}

// CreateRule 创建新规则
func CreateRule(c echo.Context) error {
	rule := new(models.Rule)
	if err := c.Bind(rule); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	rule.ID = "" // 让 GORM 自动生成 UUID

	if result := db.DB.Create(&rule); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.JSON(http.StatusCreated, rule)
}

// UpdateRule 更新规则
func UpdateRule(c echo.Context) error {
	id := c.Param("id")
	var rule models.Rule
	if result := db.DB.First(&rule, "id = ?", id); result.Error != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Rule not found")
	}

	updates := new(models.Rule)
	if err := c.Bind(updates); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// 仅允许更新特定字段，避免意外修改 ID 或创建时间
	rule.Name = updates.Name
	rule.Type = updates.Type
	rule.Match = updates.Match
	rule.Action = updates.Action
	rule.Description = updates.Description

	if result := db.DB.Save(&rule); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.JSON(http.StatusOK, rule)
}

// DeleteRule 删除规则
func DeleteRule(c echo.Context) error {
	id := c.Param("id")
	if result := db.DB.Delete(&models.Rule{}, "id = ?", id); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
