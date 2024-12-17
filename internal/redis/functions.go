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
