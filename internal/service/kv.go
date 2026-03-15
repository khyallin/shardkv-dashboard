package service

import (
	"fmt"

	"github.com/khyallin/shardkv/api"
	"github.com/khyallin/shardkv/client"

	"github.com/khyallin/shardkv-dashboard/pkg/shardkv"
)

type KVService struct {
	client *client.Clerk
}

func NewKVService() *KVService {
	skv := shardkv.New()
	return &KVService{
		client: skv.MakeClient(),
	}
}

func (s *KVService) Get(key string) (string, int, error) {
	value, version, err := s.client.Get(key)
	if err != api.OK {
		return "", 0, fmt.Errorf("KVService Get %s: %v", key, err)
	}
	return value, int(version), nil
}

func (s *KVService) Put(key, value string) error {
	_, version, err := s.client.Get(key)
	if err == api.ErrNoKey {
		version = 0
	} else if err != api.OK {
		return fmt.Errorf("KVService Get %s: %v", key, err)
	}

	err = s.client.Put(key, value, version)
	if err != api.OK {
		return fmt.Errorf("KVService Put %s: %v", key, err)
	}
	return nil
}
