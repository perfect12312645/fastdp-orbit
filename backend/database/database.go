package database

import (
	"fastdp-orbit/backend/config"
	"fastdp-orbit/backend/database/migrations"
	"fastdp-orbit/backend/pkg/logger"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

var DB *gorm.DB

// Init initializes the database connection
func Init(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Type {
	case "sqlite":
		dialector = sqlite.Open(cfg.Path)
	case "mysql":
		// TODO: Implement MySQL support
		// dialector = mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName))
	default:
		dialector = sqlite.Open(cfg.Path)
	}

	// 使用zap日志
	zapgormLogger := zapgorm2.New(logger.GetLogger())
	zapgormLogger.SetAsDefault()

	var err error
	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger: zapgormLogger,
	})
	if err != nil {
		logger.Error("数据库连接失败", zap.Error(err), zap.String("type", cfg.Type), zap.String("path", cfg.Path))
		return nil, err
	}

	logger.Info("数据库连接成功", zap.String("type", cfg.Type))

	// Run migrations
	if err := migrations.InitialMigration(DB); err != nil {
		logger.Error("数据库迁移失败", zap.Error(err))
		return nil, err
	}

	return DB, nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
