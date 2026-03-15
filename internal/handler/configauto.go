package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigAutoRequest struct {
	Auto bool `json:"auto"`
}

type ConfigAutoResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) ConfigAuto(c *gin.Context) {
	var req ConfigAutoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	err := h.configService.SetAuto(req.Auto)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	resp := ConfigAutoResponse{
		Code:    0,
		Message: "OK",
	}
	c.JSON(http.StatusOK, resp)
}
