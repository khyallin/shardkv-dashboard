package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type KVPutRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KVPutResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) KVPut(c *gin.Context) {
	var req KVPutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	err := h.kvService.Put(req.Key, req.Value)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	resp := KVPutResponse{
		Code:    0,
		Message: "OK",
	}
	c.JSON(http.StatusOK, resp)
}
