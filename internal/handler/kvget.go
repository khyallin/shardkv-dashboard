package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type KVGetRequest struct {
	Key string `json:"key"`
}

type KVGetResponse struct {
	Code    int  		    `json:"code"`
	Message string 			`json:"message"`
	Kvtype  string          `json:"type"`
	Value   json.RawMessage `json:"value"`
	Version int             `json:"version"`
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
	kvtype, value, version, err := h.kvService.Get(req.Key)
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
		Kvtype:  kvtype,
		Value:   value,
		Version: version,
	}
	c.JSON(http.StatusOK, resp)
}
