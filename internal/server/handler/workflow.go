package handler

import (
	"github.com/gin-gonic/gin"

	"myai-novel-go/internal/domain/shared"
	"myai-novel-go/internal/domain/workflows"
	"myai-novel-go/internal/server/middleware"
	"myai-novel-go/internal/workflow"
)

type WorkflowHandler struct {
	plan         *workflows.PlanWorkflow
	draft        *workflows.DraftWorkflow
	review       *workflows.ReviewWorkflow
	repair       *workflows.RepairWorkflow
	approve      *workflows.ApproveWorkflow
	stageSummary *workflows.StageSummaryWorkflow
	tasks        *workflow.Service
}

func NewWorkflowHandler(
	plan *workflows.PlanWorkflow, draft *workflows.DraftWorkflow,
	review *workflows.ReviewWorkflow, repair *workflows.RepairWorkflow,
	approve *workflows.ApproveWorkflow, stageSummary *workflows.StageSummaryWorkflow,
	tasks *workflow.Service,
) *WorkflowHandler {
	return &WorkflowHandler{plan: plan, draft: draft, review: review, repair: repair, approve: approve, stageSummary: stageSummary, tasks: tasks}
}

func (h *WorkflowHandler) Register(r *gin.RouterGroup) {
	r.POST("/api/workflows/plan", h.runPlan)
	r.POST("/api/workflows/draft", h.runDraft)
	r.POST("/api/workflows/review", h.runReview)
	r.POST("/api/workflows/repair", h.runRepair)
	r.POST("/api/workflows/approve", h.runApprove)
	r.POST("/api/workflows/stage-summary", h.runStageSummary)
	r.POST("/api/workflows/author-intent", h.runAuthorIntent)

	r.POST("/api/workflows/plan/tasks", h.startPlan)
	r.POST("/api/workflows/draft/tasks", h.startDraft)
	r.POST("/api/workflows/review/tasks", h.startReview)
	r.POST("/api/workflows/repair/tasks", h.startRepair)
	r.POST("/api/workflows/approve/tasks", h.startApprove)
	r.POST("/api/workflows/author-intent/tasks", h.startAuthorIntent)
	r.GET("/api/workflow-tasks/:taskId", h.getTask)
	r.POST("/api/workflow-tasks/:taskId/terminate", h.terminateTask)
	r.GET("/api/books/:bookId/chapters/:chapterNo/workflow-tasks", h.listTasks)
	r.GET("/api/books/:bookId/chapters/:chapterNo/workflow-tasks/latest", h.getLatestTask)
}

func (h *WorkflowHandler) runPlan(c *gin.Context) {
	var in workflows.PlanInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.plan.Run(c.Request.Context(), in, workflows.NoopNotifier)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, res)
}

func (h *WorkflowHandler) runDraft(c *gin.Context) {
	var in workflows.DraftInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.draft.Run(c.Request.Context(), in, workflows.NoopNotifier)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, res)
}

func (h *WorkflowHandler) runReview(c *gin.Context) {
	var in workflows.ReviewInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.review.Run(c.Request.Context(), in, workflows.NoopNotifier)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, res)
}

func (h *WorkflowHandler) runRepair(c *gin.Context) {
	var in workflows.RepairInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.repair.Run(c.Request.Context(), in, workflows.NoopNotifier)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, res)
}

func (h *WorkflowHandler) runApprove(c *gin.Context) {
	var in workflows.ApproveInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.approve.Run(c.Request.Context(), in, workflows.NoopNotifier)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, res)
}

func (h *WorkflowHandler) runStageSummary(c *gin.Context) {
	var in workflows.StageSummaryInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.stageSummary.Run(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, res)
}

func (h *WorkflowHandler) startPlan(c *gin.Context) {
	var in workflows.PlanInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	t, err := h.tasks.StartPlan(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	accepted(c, t)
}

func (h *WorkflowHandler) startDraft(c *gin.Context) {
	var in workflows.DraftInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	t, err := h.tasks.StartDraft(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	accepted(c, t)
}

func (h *WorkflowHandler) startReview(c *gin.Context) {
	var in workflows.ReviewInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	t, err := h.tasks.StartReview(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	accepted(c, t)
}

func (h *WorkflowHandler) startRepair(c *gin.Context) {
	var in workflows.RepairInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	t, err := h.tasks.StartRepair(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	accepted(c, t)
}

func (h *WorkflowHandler) startApprove(c *gin.Context) {
	var in workflows.ApproveInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	if in.DryRun {
		// dryRun 不入异步,直接同步执行
		res, err := h.approve.Run(c.Request.Context(), in, workflows.NoopNotifier)
		if err != nil {
			middleware.AbortWithError(c, err)
			return
		}
		ok(c, res)
		return
	}
	t, err := h.tasks.StartApprove(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	accepted(c, t)
}

func (h *WorkflowHandler) getTask(c *gin.Context) {
	id, err := middleware.ParseInt64Param(c, "taskId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	t, err := h.tasks.Get(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, t)
}

func (h *WorkflowHandler) runAuthorIntent(c *gin.Context) {
	var in workflows.AuthorIntentInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	res, err := h.plan.GenerateAuthorIntent(c.Request.Context(), in, workflows.NoopNotifier)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, res)
}

func (h *WorkflowHandler) startAuthorIntent(c *gin.Context) {
	var in workflows.AuthorIntentInput
	if err := bind(c, &in); err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	t, err := h.tasks.StartAuthorIntent(c.Request.Context(), in)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	accepted(c, t)
}

func (h *WorkflowHandler) terminateTask(c *gin.Context) {
	id, err := middleware.ParseInt64Param(c, "taskId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	t, err := h.tasks.Terminate(c.Request.Context(), id)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, t)
}

func (h *WorkflowHandler) listTasks(c *gin.Context) {
	bookID, err := middleware.ParseInt64Param(c, "bookId")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	cn, err := middleware.ParseIntParam(c, "chapterNo")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	limit, err := parseLimit(c, 50, 100)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	rows, err := h.tasks.List(c.Request.Context(), bookID, cn, limit)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, rows)
}

func (h *WorkflowHandler) getLatestTask(c *gin.Context) {
	bookID, _ := middleware.ParseInt64Param(c, "bookId")
	cn, err := middleware.ParseIntParam(c, "chapterNo")
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	taskType := c.Query("type")
	if taskType == "" {
		middleware.AbortWithError(c, shared.BadRequest("missing type query param"))
		return
	}
	t, err := h.tasks.GetLatest(c.Request.Context(), bookID, cn, taskType)
	if err != nil {
		middleware.AbortWithError(c, err)
		return
	}
	ok(c, t)
}
