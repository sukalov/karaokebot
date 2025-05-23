package common

import (
	"encoding/json"
	"fmt"
	"sort"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/state"
	"github.com/sukalov/karaokebot/internal/utils"
)

type CommonHandlers struct {
	userManager *state.StateManager
}

func GetCommandHandlers(userManager *state.StateManager) map[string]func(b *bot.Bot, update tgbotapi.Update) error {
	handlers := newCommonHandlers(userManager)
	return map[string]func(b *bot.Bot, update tgbotapi.Update) error{
		"line":  handlers.lineHandler,
		"users": handlers.usersHandler,
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
func GetCallbackHandlers(userManager *state.StateManager) map[string]func(b *bot.Bot, update tgbotapi.Update) error {
	return map[string]func(b *bot.Bot, update tgbotapi.Update) error{}
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

	sort.Sort(state.ByTimeAdded(lineUsers))

	lineMessage := ""
	i := 0
	for idx, userState := range lineUsers {
		var note string
		if userState.SongNote == "" {
			note = ""
		} else {
			note = fmt.Sprintf("   заметка по песне: %s\n", userState.SongNote)
		}
		lineMessage += fmt.Sprintf(
			"%d. %s\n   песня: [%s](%s)\n   добавлен: %s\n   юзернейм: @%s\n%s\n",
			idx+1,
			userState.TypedName,
			userState.SongName,
			userState.SongLink,
			utils.ConvertToMoscowTime(userState.TimeAdded),
			userState.Username,
			note,
		)
		i += 1
		if i == 20 {
			b.SendMessageWithMarkdown(message.Chat.ID, lineMessage, true)
			i = 0
			lineMessage = ""
		}
	}

	return b.SendMessageWithMarkdown(message.Chat.ID, lineMessage, true)
}

func (h *CommonHandlers) usersHandler(b *bot.Bot, update tgbotapi.Update) error {

	userStates := h.userManager.GetAll()

	jsonData, err := json.MarshalIndent(userStates, "", "  ")
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, "failed to convert user states to JSON")
	}

	jsonMessage := string(jsonData)
	return b.SendMessageWithMarkdown(update.Message.Chat.ID, "```json\n"+jsonMessage+"\n```", false)
}
