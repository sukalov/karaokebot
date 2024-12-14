package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/bot/common"
	"github.com/sukalov/karaokebot/internal/songbook"
	"github.com/sukalov/karaokebot/internal/state"
	"github.com/sukalov/karaokebot/internal/users"
)

type ClientHandlers struct {
	userManager *state.StateManager
}

func NewClientHandlers(userManager *state.StateManager) *ClientHandlers {
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
		for _, state := range h.userManager.GetAll() {
			fmt.Println(state)
			if state.Username == message.From.UserName && state.Stage == users.StageAskingName {
				state.SongID = songID
				state.SongName = songbook.FormatSongName(song)
				state.SongLink = song.Link
				h.userManager.EditState(context.Background(), state.ID, state)

				return b.SendMessageWithMarkdown(
					message.Chat.ID,
					fmt.Sprintf("привет! *как тебя зовут?* \n\n (чтобы записаться и спеть песню \"%s\" осталось только написать имя певца/певцов)", state.SongName),
					false,
				)
			}
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
		ctx := context.Background()
		h.userManager.AddUser(ctx, userState)
		return b.SendMessageWithMarkdown(
			message.Chat.ID,
			fmt.Sprintf("привет! *как тебя зовут?* \n\n (чтобы записаться и спеть песню \"%s\" осталось только написать имя певца/певцов)", userState.SongName),
			false,
		)
	}
	return b.SendMessage(
		message.Chat.ID,
		"не, просто так не работает. выбор песен в сонгбуке: https://karaoke.sukalov.dev",
	)
}

func (h *ClientHandlers) nameHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message
	typedName := message.Text

	userStates := h.userManager.GetAll()

	var stateToUpdate *users.UserState
	for i, state := range userStates {
		if state.Stage == users.StageAskingName && state.Username == message.From.UserName {
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
	ctx := context.Background()
	h.userManager.EditState(ctx, stateToUpdate.ID, *stateToUpdate)
	h.userManager.Sync(ctx)

	return b.SendMessageWithMarkdown(
		message.Chat.ID,
		fmt.Sprintf("отлично, %s! вы выбрали песню \"%s\". скоро вас позовут на сцену\n\nа слова можно найти [здесь](%s)",
			typedName, stateToUpdate.SongName, stateToUpdate.SongLink),
		false,
	)
}

func (h *ClientHandlers) usersHandler(b *bot.Bot, update tgbotapi.Update) error {

	// Convert the map of states to a slice for JSON marshaling
	userStates := h.userManager.GetAll()

	jsonData, err := json.MarshalIndent(userStates, "", "  ")
	if err != nil {
		// If JSON marshaling fails, send an error message
		return b.SendMessage(update.Message.Chat.ID, "failed to convert user states to JSON")
	}

	// Convert JSON bytes to string for sending
	jsonMessage := string(jsonData)

	// Send the JSON message to the Telegram bot
	return b.SendMessageWithMarkdown(update.Message.Chat.ID, "```json\n"+jsonMessage+"\n```", false)
}

func SetupHandlers(clientBot *bot.Bot, userManager *state.StateManager) {
	handlers := NewClientHandlers(userManager)
	messageHandlers := []func(b *bot.Bot, update tgbotapi.Update) error{
		func(b *bot.Bot, update tgbotapi.Update) error {
			if update.Message == nil {
				return nil
			}
			// Handle name input for song selection
			userStates := userManager.GetAll()
			for _, state := range userStates {
				if state.Stage == users.StageAskingName && state.Username == update.Message.From.UserName {
					return handlers.nameHandler(b, update)
				}
			}
			return nil
		},
	}

	// Pass userManager to GetCommandHandlers
	commandHandlers := common.GetCommandHandlers(userManager)
	commandHandlers["start"] = handlers.startHandler
	commandHandlers["users"] = handlers.usersHandler

	callbackHandlers := common.GetCallbackHandlers()
	go clientBot.Start(
		commandHandlers,
		messageHandlers,
		callbackHandlers,
	)
}
