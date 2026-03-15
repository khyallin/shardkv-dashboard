package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type GroupCreateRequest struct{}

type GroupCreateResponse struct {
	Code    int 	`json:"code"`
	Message string  `json:"message"`
	Gid     int		`json:"gid"`
}

func (h *Handler) GroupCreate(c *gin.Context) {
	var req GroupCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	gid, err := h.configService.CreateGroup()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	resp := GroupCreateResponse{
		Code:    0,
		Message: "OK",
		Gid:     gid,
	}
	c.JSON(http.StatusOK, resp)
}
