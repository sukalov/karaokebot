package users

import (
	"sync"
	"time"
)

type UserState struct {
	Username  string    `json:"username"`
	TgName    string    `json:"tg_name"`
	SongID    string    `json:"song_id"`
	SongName  string    `json:"song_name"`
	SongLink  string    `json:"song_link"`
	Stage     string    `json:"stage"`
	TimeAdded time.Time `json:"time_added"`
	TypedName string    `json:"typed_name"`
}

// UserStateManager manages user states with thread-safety
type UserStateManager struct {
	states map[int64]UserState
	mu     sync.RWMutex
}

const (
	StageAskingName = "asking_name"
	StageInLine     = "in_line"
)

// NewUserStateManager creates a new UserStateManager
func NewUserStateManager() *UserStateManager {
	return &UserStateManager{
		states: make(map[int64]UserState),
	}
}

// Get retrieves a user's state
func (m *UserStateManager) Get(userID int64) (UserState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, exists := m.states[userID]
	return state, exists
}

// Set updates a user's state
func (m *UserStateManager) Set(userID int64, state UserState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[userID] = state
}

// Delete removes a user's state
func (m *UserStateManager) Delete(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.states, userID)
}

// GetAllInLine returns all users currently in line
func (m *UserStateManager) GetAllInLine() map[int64]UserState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inLineUsers := make(map[int64]UserState)
	for userID, state := range m.states {
		if state.Stage == "in_line" {
			inLineUsers[userID] = state
		}
	}
	return inLineUsers
}

// Clear removes all user states
func (m *UserStateManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states = make(map[int64]UserState)
}
