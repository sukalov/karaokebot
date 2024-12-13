package common

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/state"
)

type CommonHandlers struct {
	userManager *state.StateManager
}

func GetCommandHandlers(userManager *state.StateManager) map[string]func(b *bot.Bot, update tgbotapi.Update) error {
	handlers := newCommonHandlers(userManager)
	return map[string]func(b *bot.Bot, update tgbotapi.Update) error{
		"line": handlers.lineHandler,
	}
}

// GetMessageHandlers returns common message handlers
func GetMessageHandlers() []func(b *bot.Bot, update tgbotapi.Update) error {
	return []func(b *bot.Bot, update tgbotapi.Update) error{
		func(b *bot.Bot, update tgbotapi.Update) error {
			// Common message logging or processing
			return nil
		},
	}
}

// GetCallbackHandlers returns common callback handlers
func GetCallbackHandlers() map[string]func(b *bot.Bot, update tgbotapi.Update) error {
	return map[string]func(b *bot.Bot, update tgbotapi.Update) error{
		// Common callback handlers
	}
}

func newCommonHandlers(userManager *state.StateManager) *CommonHandlers {
	return &CommonHandlers{
		userManager: userManager,
	}
}

func (h *CommonHandlers) lineHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message
	lineUsers := h.userManager.GetAllInLine()

	if len(lineUsers) == 0 {
		return b.SendMessage(message.Chat.ID, "в очереди никого нет")
	}

	lineMessage := "очередь:\n\n"
	for idx, userState := range lineUsers {
		lineMessage += fmt.Sprintf(
			"%d. %s\n   песня: [%s](%s)\n   добавлен: %s\n   юзернейм: @%s\n\n",
			idx+1,
			userState.TypedName,
			userState.SongName,
			userState.SongLink,
			userState.TimeAdded.Format("15:04:05"),
			userState.Username,
		)
	}

	return b.SendMessageWithMarkdown(message.Chat.ID, lineMessage)
}
