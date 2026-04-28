package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigGetRequest struct{}

type ConfigGetResponse struct {
	Code int       `json:"code"`
	Message string `json:"message"`
	Auto bool      `json:"auto"`
	Mode string    `json:"mode"`
}

func (h *Handler) ConfigGet(c *gin.Context) {
	var req ConfigGetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	auto, mode, err := h.configService.GetConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	resp := ConfigGetResponse{
		Code:    0,
		Message: "OK",
		Auto: auto,
		Mode: mode,
	}
	c.JSON(http.StatusOK, resp)
}
