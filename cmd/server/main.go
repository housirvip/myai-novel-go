package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/db"
	"myai-novel-go/internal/logger"
	"myai-novel-go/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	zlog, err := logger.New(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		log.Fatalf("logger init failed: %v", err)
	}
	defer zlog.Sync()

	zlog.Info("config.loaded",
		zap.String("dbClient", string(cfg.DBClient)),
		zap.String("llmProvider", string(cfg.LLMProvider)),
		zap.String("addr", cfg.ServerHost),
		zap.Int("port", cfg.ServerPort),
		zap.Int("workflowConcurrency", cfg.WorkflowMaxConcurrency),
	)

	gdb, err := db.Open(cfg)
	if err != nil {
		zlog.Fatal("db.open_failed", zap.Error(err))
	}
	zlog.Info("db.connected", zap.String("client", string(cfg.DBClient)))

	if err := db.Migrate(gdb); err != nil {
		zlog.Fatal("db.migrate_failed", zap.Error(err))
	}
	zlog.Info("db.migrate.ok")

	srv := server.New(cfg, zlog, gdb)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if recovered, err := srv.TaskService().RecoverInterrupted(rootCtx); err != nil {
		zlog.Error("workflow.task.recover_failed", zap.Error(err))
	} else if recovered > 0 {
		zlog.Warn("workflow.task.recovered", zap.Int64("count", recovered))
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		zlog.Info("server.signal_received")
		cancel()
	}()

	if err := srv.Run(rootCtx); err != nil {
		zlog.Fatal("server.run_failed", zap.Error(err))
	}
	zlog.Info("server.stopped")
}
