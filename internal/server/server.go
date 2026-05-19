package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"myai-novel-go/internal/config"
	"myai-novel-go/internal/domain/auth"
	"myai-novel-go/internal/domain/book"
	"myai-novel-go/internal/domain/chapter"
	"myai-novel-go/internal/domain/character"
	"myai-novel-go/internal/domain/faction"
	"myai-novel-go/internal/domain/item"
	"myai-novel-go/internal/domain/outline"
	"myai-novel-go/internal/domain/planning"
	"myai-novel-go/internal/domain/relation"
	"myai-novel-go/internal/domain/story_hook"
	usersettings "myai-novel-go/internal/domain/user_settings"
	"myai-novel-go/internal/domain/workflows"
	"myai-novel-go/internal/domain/world_setting"
	"myai-novel-go/internal/llmfactory"
	"myai-novel-go/internal/server/handler"
	"myai-novel-go/internal/server/middleware"
	"myai-novel-go/internal/server/webui"
	"myai-novel-go/internal/workflow"
)

type Server struct {
	cfg    *config.Config
	logger *zap.Logger
	gdb    *gorm.DB

	httpServer *http.Server
	runner     *workflow.Runner
	taskSvc    *workflow.Service
}

func New(cfg *config.Config, logger *zap.Logger, gdb *gorm.DB) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	authSvc := auth.NewService(gdb, cfg)
	userSettingsSvc := usersettings.NewService(gdb)

	r.Use(middleware.RequestContext(logger))
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.ErrorResponder())
	r.Use(middleware.SessionMiddleware(authSvc, cfg.AuthCookieName))
	r.Use(middleware.RequireBookAccess(gdb))
	r.MaxMultipartMemory = int64(cfg.ServerBodyLimit)

	root := &r.RouterGroup
	handler.RegisterHealth(root, cfg, gdb)

	// services
	bookSvc := book.NewService(gdb)
	chapterSvc := chapter.NewService(gdb)
	outlineSvc := outline.NewService(gdb)
	wsSvc := world_setting.NewService(gdb)
	charSvc := character.NewService(gdb)
	facSvc := faction.NewService(gdb)
	relSvc := relation.NewService(gdb)
	itemSvc := item.NewService(gdb)
	hookSvc := story_hook.NewService(gdb)

	// llm + retrieval + workflows
	llmF := llmfactory.New(cfg)
	embClient := llmF.NewEmbedding()
	retrievalSvc := planning.NewRetrievalServiceWithEmbedding(gdb, cfg, embClient)
	planWF := workflows.NewPlanWorkflow(gdb, cfg, llmF, retrievalSvc)
	draftWF := workflows.NewDraftWorkflow(gdb, cfg, llmF)
	reviewWF := workflows.NewReviewWorkflow(gdb, cfg, llmF)
	repairWF := workflows.NewRepairWorkflow(gdb, cfg, llmF)
	approveWF := workflows.NewApproveWorkflow(gdb, cfg, llmF)
	stageWF := workflows.NewStageSummaryWorkflow(gdb, cfg, llmF)

	runner := workflow.NewRunner(logger, cfg.WorkflowMaxConcurrency)
	taskSvc := workflow.NewService(gdb, runner, logger, planWF, draftWF, reviewWF, repairWF, approveWF, stageWF)

	// handlers
	handler.NewBookHandler(bookSvc).Register(root)
	handler.NewChapterHandler(chapterSvc).Register(root)
	(&handler.ResourceHandlers{
		Outline: outlineSvc, WorldSetting: wsSvc, Character: charSvc,
		Faction: facSvc, Relation: relSvc, Item: itemSvc, StoryHook: hookSvc,
	}).Register(root)
	handler.NewWorkflowHandler(planWF, draftWF, reviewWF, repairWF, approveWF, stageWF, taskSvc).Register(root)
	handler.NewEmbeddingHandler(gdb, retrievalSvc).Register(root)
	handler.NewAuthHandler(cfg, authSvc).Register(root)
	handler.NewUserSettingsHandler(cfg, userSettingsSvc).Register(root)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "Route not found: " + c.Request.Method + " " + c.Request.URL.Path}})
	})

	// WebUI 静态托管:启用时会覆盖上面的 NoRoute,接管 / 与 /app/* SPA fallback。
	webui.Register(r, cfg.WebUIDistPath, logger)

	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return &Server{cfg: cfg, logger: logger, gdb: gdb, httpServer: httpServer, runner: runner, taskSvc: taskSvc}
}

func (s *Server) TaskService() *workflow.Service { return s.taskSvc }

// Handler 暴露内部 gin engine,供 httptest 使用。
func (s *Server) Handler() http.Handler { return s.httpServer.Handler }

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server.starting", zap.String("addr", s.httpServer.Addr))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("server.shutdown.requested")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.ShutdownTimeoutSec)*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("server.shutdown.error", zap.Error(err))
		}
		s.runner.Shutdown(shutdownCtx)
		return nil
	}
}
