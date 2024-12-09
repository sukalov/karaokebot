package client

import (
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

func (h *ClientHandlers) StartHandler(b *bot.Bot, update tgbotapi.Update) error {
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
			Stage:    users.StageAskingName,
		}
		h.userManager.Set(message.From.ID, userState)
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

func (h *ClientHandlers) NameHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message
	userID := message.From.ID
	typedName := message.Text
	state, exists := h.userManager.Get(userID)
	if !exists || state.Stage != users.StageAskingName {
		return nil
	}
	// Update user state
	state.TypedName = typedName
	state.Stage = users.StageInLine
	state.TimeAdded = time.Now()
	h.userManager.Set(userID, state)
	return b.SendMessageWithMarkdown(
		message.Chat.ID,
		fmt.Sprintf("отлично, %s! вы выбрали песню \"%s\". скоро вас позовут на сцену\n\nа слова можно найти [здесь](%s)",
			typedName, state.SongName, state.SongLink),
	)
}

func (h *ClientHandlers) ExitHandler(b *bot.Bot, update tgbotapi.Update) error {
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

func SetupHandlers(clientBot *bot.Bot, userManager *users.UserStateManager) {
	handlers := NewClientHandlers(userManager)
	messageHandlers := []func(b *bot.Bot, update tgbotapi.Update) error{
		func(b *bot.Bot, update tgbotapi.Update) error {
			if update.Message == nil {
				return nil
			}
			// Handle name input for song selection
			if state, exists := userManager.Get(update.Message.From.ID); exists && state.Stage == users.StageAskingName {
				return handlers.NameHandler(b, update)
			}
			return nil
		},
	}

	// Pass userManager to GetCommandHandlers
	commandHandlers := common.GetCommandHandlers(userManager)
	commandHandlers["start"] = handlers.StartHandler
	commandHandlers["exit"] = handlers.ExitHandler

	callbackHandlers := common.GetCallbackHandlers()
	go clientBot.Start(
		commandHandlers,
		messageHandlers,
		callbackHandlers,
	)
}
