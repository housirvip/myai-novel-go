package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"myai-novel-go/internal/config"
)

func newDBCmd() *cobra.Command {
	c := &cobra.Command{Use: "db", Short: "数据库管理"}
	c.AddCommand(
		&cobra.Command{
			Use:   "init",
			Short: "运行迁移并初始化数据库",
			RunE: withContainer(func(_ context.Context, c *Container, _ *cobra.Command, _ []string) error {
				fmt.Println("db.init.ok client=", c.Cfg.DBClient)
				return nil
			}),
		},
		&cobra.Command{
			Use:   "migrate",
			Short: "重新跑一次 AutoMigrate(等价 init,但不打 init 字样)",
			RunE: withContainer(func(_ context.Context, c *Container, _ *cobra.Command, _ []string) error {
				fmt.Println("db.migrate.ok client=", c.Cfg.DBClient)
				return nil
			}),
		},
		&cobra.Command{
			Use:   "check",
			Short: "检查数据库连通性",
			RunE: withContainer(func(_ context.Context, c *Container, _ *cobra.Command, _ []string) error {
				sqlDB, err := c.DB.DB()
				if err != nil {
					return err
				}
				if err := sqlDB.Ping(); err != nil {
					return err
				}
				fmt.Println("db.check.ok client=", c.Cfg.DBClient)
				return nil
			}),
		},
		&cobra.Command{
			Use:   "reset",
			Short: "(SQLite 限定)删除 SQLite 文件",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				if cfg.DBClient != config.DBClientSQLite {
					return fmt.Errorf("db reset 仅支持 sqlite,当前 DB_CLIENT=%s", cfg.DBClient)
				}
				path := cfg.DBSQLitePath
				if path == "" {
					return fmt.Errorf("DB_SQLITE_PATH 为空")
				}
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					return err
				}
				fmt.Println("db.reset.ok path=", path)
				return nil
			},
		},
	)
	return c
}
