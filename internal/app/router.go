package app

import (
	"github.com/gin-gonic/gin"
	"github.com/khyallin/shardkv-dashboard/internal/handler"
)

func registerRoutes(r *gin.Engine) {
	h := handler.New()

	r.GET("/ping", h.Ping)

	api := r.Group("/api/v1")
	{
		// 获取集群配置信息
		api.GET("/groups", h.ConfigGet)
		// 新建分片组
		api.POST("/group/create", h.GroupCreate)
		// 停止分片组
		api.POST("/group/stop", h.GroupStop)

		// 移动分片
		api.POST("/config", h.ShardMove)
		// 设置自动负载均衡
		api.POST("/config/auto", h.ConfigAuto)

		// 查询键值对
		api.GET("/kv", h.KVGet)
		// 设置键值对
		api.POST("/kv", h.KVPut)
	}
}
