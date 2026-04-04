package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type StatusRequest struct {
	Gid int `json:"gid"`
}

type StatusResponse struct {
	Code       int           `json:"code"`
	Message    string        `json:"message"`
	TotalQPS   float64       `json:"total_qps"`
	DoneQPS    float64       `json:"done_qps"`
	SuccessQPS float64       `json:"success_qps"`
	MaxLatency time.Duration `json:"max_latency"`
	AvgLatency time.Duration `json:"avg_latency"`
}

func (h *Handler) GroupStatus(c *gin.Context) {
	var req StatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, StatusResponse{
			Code:    1,
			Message: err.Error(),
		})
		return
	}
	totalQPS, doneQPS, successQPS, maxLatency, avgLatency, err := h.configService.GroupStatus(req.Gid)
	if err != nil {
		c.JSON(http.StatusOK, StatusResponse{
			Code:    1,
			Message: err.Error(),
		})
		return
	}
	resp := StatusResponse{
		Code:       0,
		Message:    "OK",
		TotalQPS:   totalQPS,
		DoneQPS:    doneQPS,
		SuccessQPS: successQPS,
		MaxLatency: maxLatency,
		AvgLatency: avgLatency,
	}
	c.JSON(http.StatusOK, resp)
}
