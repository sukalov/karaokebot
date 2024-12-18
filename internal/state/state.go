package state

import (
	"context"
	"fmt"
	"sync"

	"github.com/sukalov/karaokebot/internal/redis"
	"github.com/sukalov/karaokebot/internal/users"
)

type StateManager struct {
	mu   sync.RWMutex
	list []users.UserState
}

func NewStateManager() *StateManager {
	return &StateManager{
		list: []users.UserState{},
	}
}

func (sm *StateManager) Init() error {
	ctx := context.Background()
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Retrieve the list from Redis
	list, err := redis.GetList(ctx)
	if err != nil {
		return err
	}
	sm.list = list
	return nil
}

func (sm *StateManager) AddUser(ctx context.Context, state users.UserState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.list = append(sm.list, state)
	if err := redis.SetList(ctx, sm.list); err != nil {
		fmt.Printf("error happened while adding to redis list: %s", err)
		return err
	}
	return nil
}

func (sm *StateManager) GetAll() []users.UserState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.list
}

func (sm *StateManager) GetAllThisUser(chatID int64) []users.UserState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var thisUserStates []users.UserState
	for _, state := range sm.list {
		if state.ChatID == chatID {
			thisUserStates = append(thisUserStates, state)
		}
	}
	return thisUserStates
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
	if err := redis.SetList(ctx, sm.list); err != nil {
		fmt.Printf("error happened while clearing the redis list: %s", err)
	}
	return nil
}

func (sm *StateManager) EditState(ctx context.Context, stateID int, newState users.UserState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, state := range sm.list {
		if state.ID == stateID {
			sm.list[i] = newState
			return nil
		}
	}

	return fmt.Errorf("state with ID %d not found", stateID)
}

func (sm *StateManager) Sync(ctx context.Context) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if err := redis.SetList(ctx, sm.list); err != nil {
		fmt.Printf("Error happened while updating the redis list: %s", err)
	}
}
