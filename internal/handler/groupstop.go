package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type GroupStopRequest struct {
	Gid int `json:"gid"`
}

type GroupStopResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) GroupStop(c *gin.Context) {
	var req GroupStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	err := h.configService.StopGroup(req.Gid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	resp := GroupStopResponse{
		Code:    0,
		Message: "OK",
	}
	c.JSON(http.StatusOK, resp)
}
