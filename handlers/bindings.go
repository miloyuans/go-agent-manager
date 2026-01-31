package handlers

import (
	"net/http"
	"time"

	"go-agent-manager/db"
	"go-agent-manager/models"

	"github.com/labstack/echo/v4"
)

// GetBindings 获取所有用户设备绑定
func GetBindings(c echo.Context) error {
	var bindings []models.UserDeviceBinding
	// 可以在这里 preload Device 信息以便前端显示
	if result := db.DB.Preload("Device").Find(&bindings); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}

	// 为了前端显示方便，手动填充 Device Hostname
	type BindingWithDevice struct {
		models.UserDeviceBinding
		DeviceHostname string `json:"device_hostname"`
	}
	var bindingsWithHostnames []BindingWithDevice
	for _, b := range bindings {
		bd := BindingWithDevice{
			UserDeviceBinding: b,
		}
		var device models.Device
		if err := db.DB.First(&device, "id = ?", b.DeviceID).Error; err == nil {
			bd.DeviceHostname = device.Hostname
		} else {
			bd.DeviceHostname = "未知设备"
		}
		bindingsWithHostnames = append(bindingsWithHostnames, bd)
	}

	return c.JSON(http.StatusOK, bindingsWithHostnames)
}

// CreateBinding 创建新的用户设备绑定
func CreateBinding(c echo.Context) error {
	binding := new(models.UserDeviceBinding)
	if err := c.Bind(binding); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// 验证 KeycloakUserID 和 DeviceID 是否存在
	var device models.Device
	if result := db.DB.First(&device, "id = ?", binding.DeviceID); result.Error != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid DeviceID")
	}
	// TODO: 验证 KeycloakUserID 是否为 Keycloak 中的真实用户 (可选，但推荐)

	binding.ID = "" // 让 GORM 自动生成 UUID
	binding.BoundAt = time.Now()
	binding.Status = "active" // 默认激活

	if result := db.DB.Create(&binding); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.JSON(http.StatusCreated, binding)
}

// DeleteBinding 删除用户设备绑定 (解绑)
func DeleteBinding(c echo.Context) error {
	id := c.Param("id")
	if result := db.DB.Delete(&models.UserDeviceBinding{}, "id = ?", id); result.Error != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, result.Error.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// TODO: UpdateBindingStatus 用于更改绑定状态 (active/inactive/pending)
