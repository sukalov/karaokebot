package state

import (
	"context"
	"fmt"
	"sync"

	"github.com/sukalov/karaokebot/internal/redis"
	"github.com/sukalov/karaokebot/internal/users"
)

type StateManager struct {
	mu         sync.RWMutex
	list       []users.UserState
	songCounts map[string]map[string]int
	db         *redis.DBManager
}

func NewStateManager(db *redis.DBManager) *StateManager {
	return &StateManager{
		list:       []users.UserState{},
		songCounts: make(map[string]map[string]int),
		db:         db,
	}
}

func (sm *StateManager) Init(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Retrieve the list from Redis
	list, err := sm.db.GetList(ctx)
	if err != nil {
		return err
	}
	sm.list = list

	// Retrieve song counts for each user
	for _, state := range list {
		if sm.songCounts[state.Username] == nil {
			counts, err := sm.db.GetSongCounts(ctx, state.Username)
			if err != nil {
				continue // skip if error retrieving counts
			}
			sm.songCounts[state.Username] = counts
		}
	}
	return nil
}

func (sm *StateManager) AddUser(ctx context.Context, state users.UserState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.list = append(sm.list, state)
	if err := sm.db.SetList(ctx, sm.list); err != nil {
		fmt.Printf("error happened while adding to redis list: %s", err)
		return err
	}
	if err := sm.db.IncrementSongCount(ctx, state.Username, state.SongID); err != nil {
		fmt.Printf("error happened while incrementing users song-count: %s", err)
		return err
	}
	if sm.songCounts[state.Username] == nil {
		sm.songCounts[state.Username] = make(map[string]int)
	}
	sm.songCounts[state.Username][state.SongID]++
	return nil
}

func (sm *StateManager) GetAll() []users.UserState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.list
}

func (sm *StateManager) GetAllInLine() []users.UserState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var inLineUsers []users.UserState
	for _, state := range sm.list {
		if state.Stage == users.StageInLine {
			inLineUsers = append(inLineUsers, state)
		}
	}
	return inLineUsers
}

func (sm *StateManager) Clear(ctx context.Context) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	sm.list = []users.UserState{}
	if err := sm.db.SetList(ctx, sm.list); err != nil {
		fmt.Printf("error happened while clearing the redis list: %s", err)
	}
	return nil
}

func (sm *StateManager) EditState(ctx context.Context, stateID int64, newState users.UserState) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for _, state := range sm.list {
		if state.ID == stateID {
			state = newState
		}
	}
	return nil
}

func (sm *StateManager) Sync(ctx context.Context) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if err := sm.db.SetList(ctx, sm.list); err != nil {
		fmt.Printf("Error happened while updating the redis list: %s", err)
	}
}
