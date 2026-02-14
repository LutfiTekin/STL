package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Storage struct {
	client *redis.Client
}

func NewStorage(addr string) *Storage {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &Storage{client: rdb}
}

func (s *Storage) StoreDepartures(ctx context.Context, stopID, date string, departures interface{}) error {
	data, err := json.Marshal(departures)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, "stop:"+stopID+":date:"+date+":departures", data, 24*time.Hour).Err()
}

func (s *Storage) GetDepartures(ctx context.Context, stopID, date string, target interface{}) error {
	val, err := s.client.Get(ctx, "stop:"+stopID+":date:"+date+":departures").Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), target)
}
