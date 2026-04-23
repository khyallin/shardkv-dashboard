package handler

import "github.com/khyallin/shardkv-dashboard/internal/service"

type Handler struct {
	kvService     *service.KVService
	configService *service.ConfigService
	stressService *service.StressService
}

func New() *Handler {
	return &Handler{
		kvService:     service.NewKVService(),
		configService: service.NewConfigService(),
		stressService: service.NewStressService(),
	}
}
