package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type GroupGetRequest struct{}

type GroupGetResponse struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Num     int              `json:"num"`
	Shards  []int            `json:"shards"`
	Groups  map[int][]string `json:"groups"`
}

func (h *Handler) GroupGet(c *gin.Context) {
	var req GroupGetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	num, shards, groups, err := h.configService.GetGroup()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}
	resp := GroupGetResponse{
		Code:    0,
		Message: "OK",
		Num:     num,
		Shards:  shards,
		Groups:  groups,
	}
	c.JSON(http.StatusOK, resp)
}
