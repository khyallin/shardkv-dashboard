package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigSetRequest struct {
	Auto bool   `json:"auto"`
	Mode string `json:"mode"`
}

type ConfigSetResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) ConfigSet(c *gin.Context) {
	var req ConfigSetRequest
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
	if req.Mode != "" {
		err = h.configService.SetMode(req.Mode)
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	resp := ConfigSetResponse{
		Code:    0,
		Message: "OK",
	}
	c.JSON(http.StatusOK, resp)
}
