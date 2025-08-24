package state

import (
	"context"
	"fmt"
	"sync"

	"github.com/sukalov/karaokebot/internal/redis"
	"github.com/sukalov/karaokebot/internal/users"
)

type StateManager struct {
	mu    sync.RWMutex
	list  []users.UserState
	open  bool
	limit int
}

type ByTimeAdded []users.UserState

func (a ByTimeAdded) Len() int           { return len(a) }
func (a ByTimeAdded) Less(i, j int) bool { return a[i].TimeAdded.Before(a[j].TimeAdded) }
func (a ByTimeAdded) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func NewStateManager() *StateManager {
	return &StateManager{
		list:  []users.UserState{},
		open:  false,
		limit: 3,
	}
}

func (sm *StateManager) Init() error {
	ctx := context.Background()
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Retrieve the list from Redis
	list, err := redis.GetList(ctx)
	open, err2 := redis.GetOpen(ctx)
	limit, err3 := redis.GetLimit(ctx)
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	if err3 != nil {
		return err3
	}
	sm.list = list
	sm.open = open
	sm.limit = limit
	return nil
}

func (sm *StateManager) IsOpen() bool {
	return sm.open
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

func (sm *StateManager) OpenList(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.open = true
	if err := redis.SetOpen(ctx, true); err != nil {
		fmt.Printf("error happened while saving list state to redis: %s", err)
		return err
	}
	return nil
}

func (sm *StateManager) CloseList(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.open = false
	if err := redis.SetOpen(ctx, false); err != nil {
		fmt.Printf("error happened while saving list state to redis: %s", err)
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
			if err := redis.SetList(ctx, sm.list); err != nil {
				fmt.Printf("error happened while updating the redis list: %s", err)
				return err
			}
			return nil
		}
	}

	return fmt.Errorf("state with ID %d not found", stateID)
}

func (sm *StateManager) Sync(ctx context.Context) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if err := redis.SetList(ctx, sm.list); err != nil {
		fmt.Printf("error happened while updating the redis list: %s", err)
		return err
	}
	return nil
}

func (sm *StateManager) RemoveState(ctx context.Context, stateID int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	result := []users.UserState{}
	for _, state := range sm.list {
		if state.ID != stateID {
			result = append(result, state)
		}
	}
	sm.list = result
	if err := redis.SetList(ctx, result); err != nil {
		fmt.Printf("error happened while updating the redis list: %s", err)
		return err
	}
	return nil
}

func (sm *StateManager) GetLimit() int {
	return sm.limit
}

func (sm *StateManager) SetLimit(ctx context.Context, limit int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.limit = limit
	if err := redis.SetLimit(ctx, limit); err != nil {
		fmt.Printf("error happened while updating the redis limit: %s", err)
		return err
	}
	return nil
}
