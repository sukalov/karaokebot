package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	redisClient "github.com/go-redis/redis/v8"
	"github.com/sukalov/karaokebot/internal/users"
	"github.com/sukalov/karaokebot/internal/utils"
)

type DBManager struct {
	client *redisClient.Client
}

func NewDBManager() *DBManager {
	env, err := utils.LoadEnv([]string{"REDIS_URL", "REDIS_PASSWORD"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load db env %s.", err)
		os.Exit(1)
	}
	opt, _ := redisClient.ParseURL(fmt.Sprintf("rediss://default:%s@%s", env["REDIS_PASSWORD"], env["REDIS_URL"]))
	client := redisClient.NewClient(opt)

	return &DBManager{client: client}
}

// SetList sets the entire list of users.UserState to Redis
func (redis *DBManager) SetList(ctx context.Context, list []users.UserState) error {
	listJSON, err := json.Marshal(list)
	if err != nil {
		return err
	}
	return redis.client.Set(ctx, "list", listJSON, 0).Err()
}

// GetList retrieves the list of users.UserState from Redis
func (redis *DBManager) GetList(ctx context.Context) ([]users.UserState, error) {
	data, err := redis.client.Get(ctx, "list").Bytes()
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

func (redis *DBManager) IncrementSongCount(ctx context.Context, username string, songID string) error {
	err := redis.client.HIncrBy(ctx, username, songID, 1).Err()
	if err != nil {
		return fmt.Errorf("failed to increment song count for user %s and song ID %s: %v", username, songID, err)
	}
	return nil
}

// GetSongCounts retrieves the song counts for a user
func (redis *DBManager) GetSongCounts(ctx context.Context, username string) (map[string]int, error) {
	result := make(map[string]int)
	raw, err := redis.client.HGetAll(ctx, username).Result()
	if err != nil {
		if err == redisClient.Nil {
			return result, nil
		}
		return nil, err
	}
	for songID, count := range raw {
		countInt, err := strconv.Atoi(count)
		if err != nil {
			continue // skip invalid counts
		}
		result[songID] = countInt
	}
	return result, nil
}
