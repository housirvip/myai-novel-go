// migrate-from-node 把 Node 版 myai-novel 的 SQLite/MySQL 数据迁到 Go 版数据库。
// 因为两边表结构完全对齐(Go 项目就是按 Node schema 建的),迁移本质是逐表
// SELECT * → CreateInBatches。
//
// 用法:
//
//	migrate-from-node --source ../myai-novel/data/novel.db
//	    [--source-driver sqlite|mysql]
//	    [--source-dsn '...']     # mysql 时用,sqlite 用 --source 即可
//	    [--target-driver sqlite|mysql]
//	    [--target-dsn '...']     # 留空则读 .env(同 server 配置)
//	    [--batch 500]
//	    [--dry-run]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db"
	"myai-novel-go/internal/db/models"
)

func main() {
	var (
		source       = flag.String("source", "", "source SQLite path (when --source-driver=sqlite)")
		sourceDriver = flag.String("source-driver", "sqlite", "sqlite|mysql")
		sourceDSN    = flag.String("source-dsn", "", "source DSN (when --source-driver=mysql)")
		targetDriver = flag.String("target-driver", "", "sqlite|mysql (default: from .env)")
		targetDSN    = flag.String("target-dsn", "", "target DSN (default: from .env)")
		batchSize    = flag.Int("batch", 500, "batch insert size")
		dryRun       = flag.Bool("dry-run", false, "only count rows, don't insert")
	)
	flag.Parse()

	srcDB, err := openSource(*sourceDriver, *source, *sourceDSN)
	if err != nil {
		log.Fatalf("open source: %v", err)
	}
	tgtDB, err := openTarget(*targetDriver, *targetDSN)
	if err != nil {
		log.Fatalf("open target: %v", err)
	}

	// 确保目标 schema 已就位
	if err := db.Migrate(tgtDB); err != nil {
		log.Fatalf("target migrate: %v", err)
	}

	ctx := context.Background()
	for _, m := range models.All() {
		name := tableName(tgtDB, m)
		moved, err := transferTable(ctx, srcDB, tgtDB, m, *batchSize, *dryRun)
		if err != nil {
			log.Fatalf("transfer %s: %v", name, err)
		}
		fmt.Printf("  %-22s %d rows%s\n", name, moved, dryRunSuffix(*dryRun))
	}
	fmt.Println("migrate.done")
}

func openSource(driver, sqlitePath, dsn string) (*gorm.DB, error) {
	cfg := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	}
	switch driver {
	case "sqlite":
		if sqlitePath == "" {
			return nil, fmt.Errorf("--source is required when --source-driver=sqlite")
		}
		if _, err := os.Stat(sqlitePath); err != nil {
			return nil, fmt.Errorf("source sqlite not found: %s", sqlitePath)
		}
		full := sqlitePath + "?_journal_mode=WAL&_busy_timeout=5000&cache=shared&mode=ro"
		return gorm.Open(sqlite.Open(full), cfg)
	case "mysql":
		if dsn == "" {
			return nil, fmt.Errorf("--source-dsn is required when --source-driver=mysql")
		}
		return gorm.Open(mysql.Open(dsn), cfg)
	}
	return nil, fmt.Errorf("unknown source driver: %s", driver)
}

func openTarget(driver, dsn string) (*gorm.DB, error) {
	if driver == "" && dsn == "" {
		// 走 .env(等价 server 配置)
		c, err := config.Load()
		if err != nil {
			return nil, err
		}
		return db.Open(c)
	}
	cfg := &gorm.Config{
		Logger:                                   gormlogger.Default.LogMode(gormlogger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	}
	switch driver {
	case "sqlite":
		full := dsn + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&cache=shared"
		return gorm.Open(sqlite.Open(full), cfg)
	case "mysql":
		return gorm.Open(mysql.Open(dsn), cfg)
	}
	return nil, fmt.Errorf("unknown target driver: %s", driver)
}

// transferTable 通用搬运:
//  1. 反射出 model slice 类型
//  2. 源端 SELECT * 分页(按 id ASC)
//  3. 目标端 CreateInBatches —— GORM 会保留 source 行的 id(autoIncrement 字段已带值)
//
// 注意:目标表在调用前应为空,否则会有主键冲突。脚本不会自动 truncate,
// 让操作者显式删 db 文件(或先跑 `novel db reset`)。
func transferTable(ctx context.Context, src, tgt *gorm.DB, model any, batch int, dryRun bool) (int, error) {
	rt := reflect.TypeOf(model).Elem()
	sliceType := reflect.SliceOf(rt)

	const pageSize = 1000
	moved := 0
	offset := 0
	for {
		slicePtr := reflect.New(sliceType)
		// 源端查表名按 model 的 TableName() 自动解析
		if err := src.WithContext(ctx).Order("id ASC").
			Limit(pageSize).Offset(offset).Find(slicePtr.Interface()).Error; err != nil {
			return moved, err
		}
		count := slicePtr.Elem().Len()
		if count == 0 {
			break
		}
		if !dryRun {
			if err := tgt.WithContext(ctx).
				Session(&gorm.Session{FullSaveAssociations: false}).
				CreateInBatches(slicePtr.Interface(), batch).Error; err != nil {
				return moved, err
			}
		}
		moved += count
		offset += pageSize
	}
	return moved, nil
}

func tableName(db *gorm.DB, model any) string {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return reflect.TypeOf(model).Elem().Name()
	}
	return stmt.Table
}

func dryRunSuffix(dryRun bool) string {
	if dryRun {
		return " (dry-run, not inserted)"
	}
	return ""
}

// 引入 time 仅为避免 -w 警告(后续若做进度 tick 可以用)
var _ = time.Now
