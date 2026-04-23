package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/khyallin/shardkv-dashboard/internal/service"
)

type StressRunRequest struct {
	DurationSec int    `json:"duration_sec"`
	Concurrency int    `json:"concurrency"`
	ReadRatio   int    `json:"read_ratio"`
	ValueSize   int    `json:"value_size"`
	TargetGID   int    `json:"target_gid"`
	KeyPrefix   string `json:"key_prefix"`
}

type StressRunResponse struct {
	Code    int                   `json:"code"`
	Message string                `json:"message"`
	Result  *service.StressResult `json:"result,omitempty"`
}

func (h *Handler) StressRun(c *gin.Context) {
	var req StressRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, StressRunResponse{
			Code:    1,
			Message: err.Error(),
		})
		return
	}

	prefix := strings.TrimSpace(req.KeyPrefix)
	if prefix == "" {
		prefix = "stress-"
	}

	cfg := service.StressRunConfig{
		Duration:    time.Duration(req.DurationSec) * time.Second,
		Concurrency: req.Concurrency,
		ReadRatio:   req.ReadRatio,
		ValueSize:   req.ValueSize,
		TargetGID:   req.TargetGID,
		KeyPrefix:   prefix,
	}

	result, err := h.stressService.Run(cfg)
	if err != nil {
		c.JSON(http.StatusOK, StressRunResponse{
			Code:    1,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, StressRunResponse{
		Code:    0,
		Message: "OK",
		Result:  result,
	})
}
