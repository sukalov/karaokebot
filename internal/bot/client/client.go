package client

import (
	"context"
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
	ctx := context.Background()

	err := db.Users.Register(update)
	if err != nil {
		log.Printf("error registering user: %v", err)
		return b.SendMessage(message.Chat.ID, "произошла ошибка при регистрации")
	}

	// Extract song ID from /start command
	if len(text) > 7 && strings.HasPrefix(text, "/start ") {
		songID := text[7:]
		song, found := db.Songbook.FindSongByID(songID)
		if !found {
			return b.SendMessage(message.Chat.ID, "извините, песни с таким id нет")
		}

		// Check if user exists in database
		user, err := db.Users.GetByChatID(message.Chat.ID)
		if err != nil {
			log.Printf("error fetching user: %v", err)
			return b.SendMessage(message.Chat.ID, "произошла ошибка при поиске пользователя")
		}

		// Check if user has a saved name
		var savedNameText string
		if user.SavedName.Valid {
			savedNameText = user.SavedName.String
		}

		// Check existing states for this user
		for _, state := range userStates {
			if state.ChatID == message.Chat.ID && state.Stage == users.StageAskingName {
				state.SongID = songID
				state.SongName = db.Songbook.FormatSongName(song)
				state.SongLink = song.Link
				h.userManager.EditState(ctx, state.ID, state)
				h.userManager.Sync(ctx)

				// If user has a saved name, offer to use it
				if savedNameText != "" {
					return b.SendMessageWithButtons(
						message.Chat.ID,
						fmt.Sprintf("так-так. кто будет песть песню \"%s\"?\n\nнажмите на кнопку или напишите новое имя", state.SongName),
						tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("записаться как %s", savedNameText), "use_saved_name"),
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
			Username: strings.ReplaceAll(message.From.UserName, "_", "\\_"),
			TgName:   fmt.Sprintf("%s %s", message.From.FirstName, message.From.LastName),
			SongID:   songID,
			SongName: db.Songbook.FormatSongName(song),
			SongLink: song.Link,
			ChatID:   message.Chat.ID,
			Stage:    users.StageAskingName,
		}
		h.userManager.AddUser(ctx, userState)
		h.userManager.Sync(ctx)

		// If user has a saved name, offer to use it
		if savedNameText != "" {
			return b.SendMessageWithButtons(
				message.Chat.ID,
				fmt.Sprintf("так-так. кто будет песть песню \"%s\"?\n\nнажмите на кнопку или просто напишите новое имя", userState.SongName),
				tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("записаться как %s", strings.ReplaceAll(savedNameText, "\\_", "_")), "use_saved_name"),
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

	// Answer callback immediately
	callback := tgbotapi.NewCallback(query.ID, "")
	if _, err := b.Client.Request(callback); err != nil {
		log.Printf("failed to answer callback query: %v", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	// Use context with timeout for database operations
	user, err := db.Users.GetByChatID(message.Chat.ID)
	if err != nil {
		log.Printf("error getting user by chat ID: %v", err)
		return b.SendMessage(message.Chat.ID, "произошла ошибка при получении сохраненного имени")
	}

	if !user.SavedName.Valid {
		log.Printf("user saved name not found")
		return fmt.Errorf("user saved name not found")
	}

	stateToUpdate.TypedName = user.SavedName.String
	stateToUpdate.Stage = users.StageInLine
	stateToUpdate.TimeAdded = time.Now()

	if err := h.userManager.EditState(ctx, stateToUpdate.ID, *stateToUpdate); err != nil {
		log.Printf("error editing state: %v", err)
	}

	if err := h.userManager.Sync(ctx); err != nil {
		log.Printf("error syncing state: %v", err)
	}

	// Group these operations together since they're related
	errChan := make(chan error, 2)
	go func() {
		if err := db.Users.IncrementTimesPerformed(stateToUpdate.ChatID); err != nil {
			errChan <- fmt.Errorf("increment performances failed: %w", err)
		}
		errChan <- nil
	}()

	go func() {
		if err := db.Songbook.IncrementSongCounter(stateToUpdate.SongID); err != nil {
			errChan <- fmt.Errorf("increment counter failed: %w", err)
		}
		errChan <- nil
	}()

operations:
	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				log.Printf("async operation error: %v", err)
			}
		case <-ctx.Done():
			log.Printf("timeout waiting for increment operations")
			break operations
		}
	}

	return b.SendMessageWithMarkdown(
		message.Chat.ID,
		fmt.Sprintf("отлично, %s! вы выбрали песню \"%s\". скоро вас позовут на сцену\n\nа слова можно найти [здесь](%s)",
			user.SavedName.String, stateToUpdate.SongName, stateToUpdate.SongLink),
		false,
	)
}

func (h *ClientHandlers) nameHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message
	userStates := h.userManager.GetAllThisUser(update.Message.Chat.ID)

	var stateToUpdate *users.UserState
	for i, state := range userStates {
		if state.Stage == users.StageAskingName {
			stateToUpdate = &userStates[i]
			break
		}
	}

	// If no matching state found, return
	if stateToUpdate == nil {
		return fmt.Errorf("no state with asking_name was found")
	}

	// Update the found state
	stateToUpdate.TypedName = strings.ReplaceAll(message.Text, "_", "\\_")
	stateToUpdate.Stage = users.StageInLine
	stateToUpdate.TimeAdded = time.Now()

	// Edit the state in the user manager
	ctx := context.Background()
	h.userManager.EditState(ctx, stateToUpdate.ID, *stateToUpdate)
	h.userManager.Sync(ctx)
	if err := db.Users.IncrementTimesPerformed(stateToUpdate.ChatID); err != nil {
		fmt.Printf("increment performances failed: %s", err)
	}
	if err := db.Songbook.IncrementSongCounter(stateToUpdate.SongID); err != nil {
		fmt.Printf("increment counter failed: %s", err)
	}
	if err := db.Users.UpdateSavedName(stateToUpdate.ChatID, stateToUpdate.TypedName); err != nil {
		fmt.Printf("increment counter failed: %s", err)
	}
	return b.SendMessageWithMarkdown(
		message.Chat.ID,
		fmt.Sprintf("отлично, %s! вы выбрали песню \"%s\". скоро вас позовут на сцену\n\nа слова можно найти [здесь](%s)",
			stateToUpdate.TypedName, stateToUpdate.SongName, stateToUpdate.SongLink),
		false,
	)
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
				if state.Stage == users.StageAskingName && state.ChatID == update.Message.Chat.ID {
					return handlers.nameHandler(b, update)
				}
			}

			return randomMessageHandler(b, update)
		},
	}

	// Pass userManager to GetCommandHandlers
	commandHandlers := common.GetCommandHandlers(userManager)
	commandHandlers["start"] = handlers.startHandler

	callbackHandlers := common.GetCallbackHandlers(userManager)
	callbackHandlers["use_saved_name"] = handlers.useSavedNameHandler

	go clientBot.Start(
		commandHandlers,
		messageHandlers,
		callbackHandlers,
	)
}
