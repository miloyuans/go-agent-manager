package handlers

import (
	"net/http"
	"time"

	"go-agent-manager/db"
	"go-agent-manager/models"

	"github.com/labstack/echo/v4"
)

// GetDevices 获取所有设备
func GetDevices(c echo.Context) error {
	var devices []models.Device
	if result := db.DB.Find(&devices); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.JSON(http.StatusOK, devices)
}

// CreateDevice 创建新设备 (通常由 Agent 上报)
func CreateDevice(c echo.Context) error {
	device := new(models.Device)
	if err := c.Bind(device); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// 假设 UniqueHardwareID 是 Agent 提供的，其他由后端填充
	device.ID = "" // 让 GORM 自动生成 UUID
	device.LastSeenAt = time.Now()

	if result := db.DB.Create(&device); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.JSON(http.StatusCreated, device)
}

// UpdateDevice 更新设备信息 (例如更新 LastSeenAt, 或修改其他属性)
func UpdateDevice(c echo.Context) error {
	id := c.Param("id")
	var device models.Device
	if result := db.DB.First(&device, "id = ?", id); result.Error != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Device not found")
	}

	updates := new(models.Device)
	if err := c.Bind(updates); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// 只允许更新部分字段
	device.OS = updates.OS
	device.Hostname = updates.Hostname
	device.LastSeenAt = time.Now() // 每次更新也更新最后在线时间

	if result := db.DB.Save(&device); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.JSON(http.StatusOK, device)
}

// DeleteDevice 删除设备
func DeleteDevice(c echo.Context) error {
	id := c.Param("id")
	if result := db.DB.Delete(&models.Device{}, "id = ?", id); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.NoContent(http.StatusNoContent)
}
