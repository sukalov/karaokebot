package users

import (
	"sync"
	"time"
)

type UserState struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	ChatID    int64     `json:"chat_id"`
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
	states []UserState
	mu     sync.RWMutex
	nextID int64
}

const (
	StageAskingName = "asking_name"
	StageInLine     = "in_line"
)

// NewUserStateManager creates a new UserStateManager
func NewUserStateManager() *UserStateManager {
	return &UserStateManager{
		states: []UserState{},
		nextID: 1,
	}
}

// Get retrieves a user state by its internal ID
func (m *UserStateManager) Get(stateID int64) (UserState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, state := range m.states {
		if state.ID == stateID {
			return state, true
		}
	}
	return UserState{}, false
}

// GetByUserID retrieves all states for a specific user by their chat ID
func (m *UserStateManager) GetByUserID(chatID int64) []UserState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var userStates []UserState
	for _, state := range m.states {
		if state.ChatID == chatID {
			userStates = append(userStates, state)
		}
	}
	return userStates
}

// Add adds a new user state and returns its ID
func (m *UserStateManager) Add(state UserState) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	state.ID = m.nextID
	m.states = append(m.states, state)
	m.nextID++

	return state.ID
}

// Edit updates an existing user state by its ID
func (m *UserStateManager) Edit(stateID int64, updatedState UserState) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, state := range m.states {
		if state.ID == stateID {
			updatedState.ID = stateID
			m.states[i] = updatedState
			return true
		}
	}
	return false
}

// Delete removes a specific user state by its ID
func (m *UserStateManager) Delete(stateID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, state := range m.states {
		if state.ID == stateID {
			m.states = append(m.states[:i], m.states[i+1:]...)
			return true
		}
	}
	return false
}

// DeleteByUserID removes all states for a specific user by their chat ID
func (m *UserStateManager) DeleteByUserID(chatID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var remainingStates []UserState
	for _, state := range m.states {
		if state.ChatID != chatID {
			remainingStates = append(remainingStates, state)
		}
	}
	m.states = remainingStates
}

// GetAllInLine returns all users currently in line
func (m *UserStateManager) GetAllInLine() []UserState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var inLineUsers []UserState
	for _, state := range m.states {
		if state.Stage == "in_line" {
			inLineUsers = append(inLineUsers, state)
		}
	}
	return inLineUsers
}

// GetAll returns all user states
func (m *UserStateManager) GetAll() []UserState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return append([]UserState(nil), m.states...)
}

// Clear removes all user states
func (m *UserStateManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states = []UserState{}
	m.nextID = 1
}
