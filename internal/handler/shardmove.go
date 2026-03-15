package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ShardMoveRequest struct {
	Shard int `json:"shard"`
	From  int `json:"from"`
	To    int `json:"to"`
}

type ShardMoveResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) ShardMove(c *gin.Context) {
	var req ShardMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, ShardMoveResponse{
			Code:    1,
			Message: err.Error(),
		})
		return
	}
	err := h.configService.MoveShard(req.Shard, req.From, req.To)
	if err != nil {
		c.JSON(http.StatusOK, ShardMoveResponse{
			Code:    1,
			Message: err.Error(),
		})
		return
	}
	resp := ShardMoveResponse{
		Code:    0,
		Message: "OK",
	}
	c.JSON(http.StatusOK, resp)
}
