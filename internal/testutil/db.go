package testutil

import (
	"path/filepath"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"myai-novel-go/internal/db"
	dbmodels "myai-novel-go/internal/db/models"
)

// OpenTempDB 在临时目录创建 SQLite 文件并跑迁移,返回 GORM DB。
// 用临时文件而不是 :memory:,避免 GORM PrepareStmt 的连接复用 + memory 隔离问题。
func OpenTempDB(t *testing.T) *gorm.DB {
	t.Helper()
	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db") + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_synchronous=NORMAL&cache=shared"
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(gdb); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_ = dbmodels.All
	return gdb
}
