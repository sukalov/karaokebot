package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/bot/common"
	"github.com/sukalov/karaokebot/internal/db"
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
	userStates := h.userManager.GetAll()

	// First, register the user in the database
	err := db.RegisterUser(update)
	if err != nil {
		log.Printf("error registering user: %v", err)
		return b.SendMessage(message.Chat.ID, "произошла ошибка при регистрации")
	}

	// Extract song ID from /start command
	if len(text) > 7 && strings.HasPrefix(text, "/start ") {
		songID := text[7:]
		song, found := db.FindSongByID(songID)
		if !found {
			return b.SendMessage(message.Chat.ID, "извините, песни с таким id нет")
		}

		// Check if user exists in database
		user, err := db.GetUserByChatID(message.Chat.ID)
		if err != nil {
			log.Printf("Error fetching user: %v", err)
			return b.SendMessage(message.Chat.ID, "произошла ошибка при поиске пользователя")
		}

		// Check if user has a saved name
		var savedNameText string
		if user.SavedName.Valid {
			savedNameText = user.SavedName.String
		}

		// Check existing states for this user
		for _, state := range userStates {
			if state.Username == message.From.UserName && state.Stage == users.StageAskingName {
				state.SongID = songID
				state.SongName = db.FormatSongName(song)
				state.SongLink = song.Link
				h.userManager.EditState(context.Background(), state.ID, state)

				// If user has a saved name, offer to use it
				if savedNameText != "" {
					return b.SendMessageWithButtons(
						message.Chat.ID,
						fmt.Sprintf("так-так. кто будет песть песню \"%s\"?\n\nнажмите на кнопку или напишите новое имя", state.SongName),
						tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("записаться как %s", savedNameText), "use_saved_name:"+songID),
							),
						),
					)
				}

				return b.SendMessageWithMarkdown(
					message.Chat.ID,
					fmt.Sprintf("привет! *как тебя зовут?* \n\n (чтобы записаться и спеть песню \"%s\" осталось только написать имя певца/певцов)", state.SongName),
					false,
				)
			}
		}

		// If no existing state, create a new one
		previousStates := h.userManager.GetAllThisUser(message.Chat.ID)
		if len(previousStates) >= 3 {
			return b.SendMessage(message.Chat.ID, "больше трёх раз записываться нельзя\n\nУВЫ!")
		}

		// Prepare user state
		userState := users.UserState{
			ID:       len(userStates) + 1,
			Username: message.From.UserName,
			TgName:   fmt.Sprintf("%s %s", message.From.FirstName, message.From.LastName),
			SongID:   songID,
			SongName: db.FormatSongName(song),
			SongLink: song.Link,
			ChatID:   message.Chat.ID,
			Stage:    users.StageAskingName,
		}
		ctx := context.Background()
		h.userManager.AddUser(ctx, userState)

		// If user has a saved name, offer to use it
		if savedNameText != "" {
			return b.SendMessageWithButtons(
				message.Chat.ID,
				fmt.Sprintf("так-так. кто будет песть песню \"%s\"?\n\nнажмите на кнопку или просто напишите новое имя", userState.SongName),
				tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("записаться как %s", savedNameText), "use_saved_name"),
					),
				),
			)
		}

		return b.SendMessageWithMarkdown(
			message.Chat.ID,
			fmt.Sprintf("привет! *как тебя зовут?*\n\n(чтобы записаться и спеть песню \"%s\" осталось только написать имя певца/певцов)", userState.SongName),
			false,
		)
	}

	return b.SendMessage(
		message.Chat.ID,
		"не, просто так не работает. выбор песен в сонгбуке: https://karaoke.sukalov.dev",
	)
}

func (h *ClientHandlers) useSavedNameHandler(b *bot.Bot, update tgbotapi.Update) error {
	query := update.CallbackQuery
	message := query.Message
	userStates := h.userManager.GetAllThisUser(message.Chat.ID)

	var stateToUpdate *users.UserState
	for i, state := range userStates {
		if state.Stage == users.StageAskingName {
			stateToUpdate = &userStates[i]
			break
		}
	}

	if stateToUpdate == nil {
		return b.SendMessage(message.Chat.ID, "жать на ту кнопку уже поздно")
	}

	// Fetch user to get saved name
	user, err := db.GetUserByChatID(message.Chat.ID)
	if err != nil {
		return b.SendMessage(message.Chat.ID, "произошла ошибка при получении сохраненного имени")
	}

	if user.SavedName.Valid {
		stateToUpdate.TypedName = user.SavedName.String
		stateToUpdate.Stage = users.StageInLine
		h.userManager.EditState(context.Background(), stateToUpdate.ID, *stateToUpdate)
		h.userManager.Sync(context.Background())

		// Send confirmation
		return b.SendMessageWithMarkdown(
			message.Chat.ID,
			fmt.Sprintf("отлично, %s! вы выбрали песню \"%s\". скоро вас позовут на сцену\n\nа слова можно найти [здесь](%s)",
				user.SavedName.String, stateToUpdate.SongName, stateToUpdate.SongLink),
			false,
		)
	}

	return nil
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
		return fmt.Errorf("no state with asking_name was found")
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
		return b.SendMessage(update.Message.Chat.ID, "failed to convert user states to JSON")
	}

	// Convert JSON bytes to string for sending
	jsonMessage := string(jsonData)
	return b.SendMessageWithMarkdown(update.Message.Chat.ID, "```json\n"+jsonMessage+"\n```", false)
}

func randomMessageHandler(b *bot.Bot, update tgbotapi.Update) error {
	return b.SendMessage(
		update.Message.Chat.ID,
		"этого я не понимаю...\n\nвыбор песен в сонгбуке: https://karaoke.sukalov.dev",
	)
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

			return randomMessageHandler(b, update)
		},
	}

	// Pass userManager to GetCommandHandlers
	commandHandlers := common.GetCommandHandlers(userManager)
	commandHandlers["start"] = handlers.startHandler
	commandHandlers["users"] = handlers.usersHandler

	callbackHandlers := common.GetCallbackHandlers()
	callbackHandlers["use_saved_name"] = handlers.useSavedNameHandler

	go clientBot.Start(
		commandHandlers,
		messageHandlers,
		callbackHandlers,
	)
}
