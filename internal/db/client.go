package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db/models"
)

func Open(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	}

	var dial gorm.Dialector
	switch cfg.DBClient {
	case config.DBClientSQLite:
		if err := os.MkdirAll(filepath.Dir(cfg.DBSQLitePath), 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite dir: %w", err)
		}
		dsn := cfg.DBSQLitePath + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_synchronous=NORMAL&cache=shared"
		dial = sqlite.Open(dsn)
	case config.DBClientMySQL:
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC&collation=utf8mb4_unicode_ci",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
		)
		dial = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported DB_CLIENT: %s", cfg.DBClient)
	}

	gdb, err := gorm.Open(dial, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	if cfg.DBClient == config.DBClientSQLite {
		// SQLite 写并发受限,把池调到 1 写 + 多读;通过 PRAGMA WAL 提升并发,
		// 但 SetMaxOpenConns 必须 >=1,实际靠 _busy_timeout 排队。
		sqlDB.SetMaxOpenConns(cfg.DBPoolMax)
		sqlDB.SetMaxIdleConns(2)
	} else {
		sqlDB.SetMaxOpenConns(cfg.DBPoolMax)
		sqlDB.SetMaxIdleConns(cfg.DBPoolMax / 2)
	}
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	return gdb, nil
}

func Migrate(gdb *gorm.DB) error {
	return gdb.AutoMigrate(models.All()...)
}
