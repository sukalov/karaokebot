package redis

import (
	"context"
	"encoding/json"

	redisClient "github.com/go-redis/redis/v8"
	"github.com/sukalov/karaokebot/internal/users"
)

// SetList sets the entire list of users.UserState to Redis
func SetList(ctx context.Context, list []users.UserState) error {
	listJSON, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return Client.Set(ctx, "list", listJSON, 0).Err()
}

// GetList retrieves the list of users.UserState from Redis
func GetList(ctx context.Context) ([]users.UserState, error) {
	data, err := Client.Get(ctx, "list").Bytes()
	if err != nil {
		if err == redisClient.Nil {
			return []users.UserState{}, nil
		}
		return nil, err
	}
	var list []users.UserState
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func GetOpen(ctx context.Context) (bool, error) {
	data, err := Client.Get(ctx, "open").Bytes()
	if err != nil {
		if err == redisClient.Nil {
			return false, nil
		}
		return false, err
	}
	var open bool
	if err := json.Unmarshal(data, &open); err != nil {
		return false, err
	}
	return open, nil
}

func SetOpen(ctx context.Context, open bool) error {
	openJSON, err := json.Marshal(open)
	if err != nil {
		return err
	}
	return Client.Set(ctx, "open", openJSON, 0).Err()
}

func GetLimit(ctx context.Context) (int, error) {
	data, err := Client.Get(ctx, "limit").Bytes()
	if err != nil {
		if err == redisClient.Nil {
			return 0, nil
		}
		return 0, err
	}
	var limit int
	if err := json.Unmarshal(data, &limit); err != nil {
		return 0, err
	}
	return limit, nil
}

func SetLimit(ctx context.Context, limit int) error {
	limitJSON, err := json.Marshal(limit)
	if err != nil {
		return err
	}
	return Client.Set(ctx, "limit", limitJSON, 0).Err()
}

func GetPrice(ctx context.Context) (int, error) {
	data, err := Client.Get(ctx, "price").Bytes()
	if err != nil {
		if err == redisClient.Nil {
			return 0, nil
		}
		return 0, err
	}
	var price int
	if err := json.Unmarshal(data, &price); err != nil {
		return 0, err
	}
	return price, nil
}

func SetPrice(ctx context.Context, price int) error {
	priceJSON, err := json.Marshal(price)
	if err != nil {
		return err
	}
	return Client.Set(ctx, "price", priceJSON, 0).Err()
}
