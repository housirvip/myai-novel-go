package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"myai-novel-go/internal/domain/shared"
)

func ok(c *gin.Context, v any)     { c.JSON(http.StatusOK, gin.H{"data": v}) }
func created(c *gin.Context, v any) { c.JSON(http.StatusCreated, gin.H{"data": v}) }
func accepted(c *gin.Context, v any) { c.JSON(http.StatusAccepted, gin.H{"data": v}) }

func parseLimit(c *gin.Context, def, maxV int) (int, error) {
	raw := c.Query("limit")
	if raw == "" {
		return def, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, shared.BadRequest("invalid limit query param")
	}
	if n > maxV {
		n = maxV
	}
	return n, nil
}

func bind(c *gin.Context, v any) error {
	if err := c.ShouldBindJSON(v); err != nil {
		return shared.BadRequestDetails("invalid request body", err.Error())
	}
	return nil
}
