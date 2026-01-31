package config

import (
	"log"
	"os"

	"github.com/joho/godotenv" // 用于从 .env 文件加载环境变量
	"github.com/spf13/viper"
)

// Config 结构体定义了所有应用程序配置
type Config struct {
	ServerPort string `mapstructure:"SERVER_PORT"`
	DatabaseURL string `mapstructure:"DATABASE_URL"`

	Keycloak struct {
		AuthServerURL string `mapstructure:"KEYCLOAK_AUTH_SERVER_URL"`
		Realm         string `mapstructure:"KEYCLOAK_REALM"`
		AdminClientID string `mapstructure:"KEYCLOAK_ADMIN_CLIENT_ID"`     // Backend 自身调用 Keycloak Admin API 的 Client ID
		AdminClientSecret string `mapstructure:"KEYCLOAK_ADMIN_CLIENT_SECRET"` // Backend 自身调用 Keycloak Admin API 的 Client Secret
		FrontendClientID string `mapstructure:"KEYCLOAK_FRONTEND_CLIENT_ID"` // 前端认证 Client ID (用于 JWT 验证)
	} `mapstructure:"KEYCLOAK"`

	FrontendStaticPath string `mapstructure:"FRONTEND_STATIC_PATH"` // 前端静态文件路径
}

var AppConfig Config

// LoadConfig 从环境变量或 .env 文件加载配置
func LoadConfig() {
	// 尝试加载 .env 文件，如果不存在则忽略
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Printf("Error loading .env file: %v. Proceeding with environment variables.", err)
	}

	viper.AutomaticEnv() // 自动绑定环境变量

	// Server
	viper.SetDefault("SERVER_PORT", "8080")
	// Database
	viper.SetDefault("DATABASE_URL", "postgresql://user:password@localhost:5432/dbname?sslmode=disable")
	// Keycloak (请替换为您的实际配置)
	viper.SetDefault("KEYCLOAK_AUTH_SERVER_URL", "http://localhost:8080/auth")
	viper.SetDefault("KEYCLOAK_REALM", "master")
	viper.SetDefault("KEYCLOAK_ADMIN_CLIENT_ID", "admin-cli") // Keycloak 默认的 admin-cli client
	viper.SetDefault("KEYCLOAK_ADMIN_CLIENT_SECRET", "YOUR_ADMIN_CLI_SECRET")
	viper.SetDefault("KEYCLOAK_FRONTEND_CLIENT_ID", "admin-frontend-client") // 前端 Client ID

	// Frontend Static Path
	viper.SetDefault("FRONTEND_STATIC_PATH", "./frontend/dist") // 假设前端构建后的文件在 go-agent-manager/frontend/dist 目录下

	// 将配置绑定到 AppConfig 结构体
	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode config into struct, %v", err)
	}

	// 打印 Keycloak 配置（DEBUG ONLY，生产环境请勿打印敏感信息）
	log.Printf("Loaded Keycloak Auth Server URL: %s", AppConfig.Keycloak.AuthServerURL)
	log.Printf("Loaded Keycloak Realm: %s", AppConfig.Keycloak.Realm)
	log.Printf("Loaded Keycloak Admin Client ID: %s", AppConfig.Keycloak.AdminClientID)
	log.Printf("Loaded Keycloak Frontend Client ID: %s", AppConfig.Keycloak.FrontendClientID)
}
