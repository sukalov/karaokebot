package client

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/bot/common"
	"github.com/sukalov/karaokebot/internal/songbook"
	"github.com/sukalov/karaokebot/internal/users"
)

type ClientHandlers struct {
	userManager *users.UserStateManager
}

func NewClientHandlers(userManager *users.UserStateManager) *ClientHandlers {
	return &ClientHandlers{
		userManager: userManager,
	}
}

func (h *ClientHandlers) startHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message
	text := message.Text
	// Extract song ID from /start command
	if len(text) > 7 && strings.HasPrefix(text, "/start ") {
		songID := text[7:]
		song, found := songbook.FindSongByID(songID)
		if !found {
			return b.SendMessage(message.Chat.ID, "извините, песни с таким id нет")
		}
		// Prepare user state
		userState := users.UserState{
			Username: message.From.UserName,
			TgName:   fmt.Sprintf("%s %s", message.From.FirstName, message.From.LastName),
			SongID:   songID,
			SongName: songbook.FormatSongName(song),
			SongLink: song.Link,
			ChatID:   message.Chat.ID,
			Stage:    users.StageAskingName,
		}
		h.userManager.Add(userState)
		return b.SendMessageWithMarkdown(
			message.Chat.ID,
			fmt.Sprintf("привет! выбрана песня \"%s\". *как тебя зовут?* (или того, кто будет петь)", userState.SongName),
		)
	}
	return b.SendMessage(
		message.Chat.ID,
		"не, просто так не работает. выбор песен в сонгбуке: https://karaoke-songbook.netlify.app",
	)
}

func (h *ClientHandlers) nameHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message
	userID := message.From.ID
	typedName := message.Text

	// Find the state with 'asking_name' stage for this user
	userStates := h.userManager.GetByUserID(userID)
	var stateToUpdate *users.UserState
	for i, state := range userStates {
		if state.Stage == users.StageAskingName {
			stateToUpdate = &userStates[i]
			break
		}
	}

	// If no matching state found, return
	if stateToUpdate == nil {
		return nil
	}

	// Update the found state
	stateToUpdate.TypedName = typedName
	stateToUpdate.Stage = users.StageInLine
	stateToUpdate.TimeAdded = time.Now()

	// Edit the state in the user manager
	h.userManager.Edit(stateToUpdate.ID, *stateToUpdate)

	return b.SendMessageWithMarkdown(
		message.Chat.ID,
		fmt.Sprintf("отлично, %s! вы выбрали песню \"%s\". скоро вас позовут на сцену\n\nа слова можно найти [здесь](%s)",
			typedName, stateToUpdate.SongName, stateToUpdate.SongLink),
	)
}

func (h *ClientHandlers) exitHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message
	userID := message.From.ID
	state, exists := h.userManager.Get(userID)
	if !exists || state.Stage != users.StageInLine {
		return b.SendMessage(message.Chat.ID, "вас и так нет в очереди..")
	}
	// Remove user from queue
	h.userManager.Delete(userID)
	return b.SendMessage(
		message.Chat.ID,
		"ок, вычёркиваем",
	)
}
func (h *ClientHandlers) usersHandler(b *bot.Bot, update tgbotapi.Update) error {

	// Convert the map of states to a slice for JSON marshaling
	userStates := h.userManager.GetAll()

	jsonData, err := json.MarshalIndent(userStates, "", "  ")
	if err != nil {
		// If JSON marshaling fails, send an error message
		return b.SendMessage(update.Message.Chat.ID, "Failed to convert user states to JSON")
	}

	// Convert JSON bytes to string for sending
	jsonMessage := string(jsonData)

	// Send the JSON message to the Telegram bot
	return b.SendMessage(update.Message.Chat.ID, "User States:\n```json\n"+jsonMessage+"\n```")
}

func SetupHandlers(clientBot *bot.Bot, userManager *users.UserStateManager) {
	handlers := NewClientHandlers(userManager)
	messageHandlers := []func(b *bot.Bot, update tgbotapi.Update) error{
		func(b *bot.Bot, update tgbotapi.Update) error {
			if update.Message == nil {
				return nil
			}
			// Handle name input for song selection
			userStates := userManager.GetByUserID(update.Message.From.ID)
			for _, state := range userStates {
				if state.Stage == users.StageAskingName {
					return handlers.nameHandler(b, update)
				}
			}
			return nil
		},
	}

	// Pass userManager to GetCommandHandlers
	commandHandlers := common.GetCommandHandlers(userManager)
	commandHandlers["start"] = handlers.startHandler
	commandHandlers["exit"] = handlers.exitHandler
	commandHandlers["users"] = handlers.usersHandler

	callbackHandlers := common.GetCallbackHandlers()
	go clientBot.Start(
		commandHandlers,
		messageHandlers,
		callbackHandlers,
	)
}
