package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type KVPutRequest struct {
	Key     string          `json:"key"`
	Kvtype  string          `json:"type"`
	Value   json.RawMessage `json:"value"`
	Version int             `json:"version"`
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
	err := h.kvService.Put(req.Key, req.Kvtype, req.Value, req.Version)
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
