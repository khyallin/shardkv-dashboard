package service

import (
	"encoding/json"
	"fmt"
	"math"

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

type TypeValue struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

func (s *KVService) checkTypeValue(tv TypeValue) error {
	if len(tv.Value) == 0 {
		return fmt.Errorf("empty value")
	}

	switch tv.Type {
	case "string":
		var value string
		if err := json.Unmarshal(tv.Value, &value); err != nil {
			return err
		}
	case "int":
		var value int
		if err := json.Unmarshal(tv.Value, &value); err != nil {
			return err
		}
		return nil
	case "float":
		var value float64
		if err := json.Unmarshal(tv.Value, &value); err != nil {
			return err
		}
		if math.IsInf(value, 0) || math.IsNaN(value) {
			return fmt.Errorf("invalid float value")
		}
		return nil
	case "bool":
		var value bool
		if err := json.Unmarshal(tv.Value, &value); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported type: %s", tv.Type)
	}
	return nil
}

func (s *KVService) Get(key string) (string, json.RawMessage, int, error) {
	jsonValue, version, clientErr := s.client.Get(key)
	if clientErr != api.OK {
		return "", nil, 0, fmt.Errorf("KVService Get %s: %v", key, clientErr)
	}

	tv := TypeValue{}
	if err := json.Unmarshal([]byte(jsonValue), &tv); err != nil {
		return "", nil, 0, fmt.Errorf("KVService Get %s: unmarshal value: %v", key, err)
	}
	if err := s.checkTypeValue(tv); err != nil {
		return "", nil, 0, fmt.Errorf("KVService Get %s: invalid type/value: %v", key, err)
	}
	return tv.Type, tv.Value, int(version), nil
}

func (s *KVService) Put(key string, kvtype string, value json.RawMessage, version int) error {
	tv := TypeValue{
		Type:  kvtype,
		Value: value,
	}
	if err := s.checkTypeValue(tv); err != nil {
		return fmt.Errorf("KVService Put %s: invalid type/value: %v", key, err)
	}
	jsonValue, jsonErr := json.Marshal(tv)
	if jsonErr != nil {
		return fmt.Errorf("KVService Put %s: marshal value: %v", key, jsonErr)
	}

	err := s.client.Put(key, string(jsonValue), api.Tversion(version))
	if err != api.OK {
		return fmt.Errorf("KVService Put %s: %v", key, err)
	}
	return nil
}
