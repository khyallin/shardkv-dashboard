package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type KVGetRequest struct {
	Key string `json:"key"`
}

type KVGetResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Value   string `json:"value"`
	Version int    `json:"version"`
}

func (h *Handler) KVGet(c *gin.Context) {
	var req KVGetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	value, version, err := h.kvService.Get(req.Key)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	resp := KVGetResponse{
		Code:    0,
		Message: "OK",
		Value:   value,
		Version: version,
	}
	c.JSON(http.StatusOK, resp)
}
