package db

import (
	"log"

	"go-agent-manager/config"
	"go-agent-manager/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接并自动迁移模型
func InitDB() {
	var err error
	DB, err = gorm.Open(postgres.Open(config.AppConfig.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 在控制台打印 SQL 日志
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection established.")

	// 自动迁移数据库模式
	err = DB.AutoMigrate(
		&models.Device{},
		&models.UserDeviceBinding{},
		&models.Rule{},
	)
	if err != nil {
		log.Fatalf("Failed to auto migrate database: %v", err)
	}

	log.Println("Database auto-migration completed.")
}
