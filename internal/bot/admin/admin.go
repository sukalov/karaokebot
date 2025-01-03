package admin

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/bot/common"
	"github.com/sukalov/karaokebot/internal/state"
)

type AdminHandlers struct {
	userManager     *state.StateManager
	admins          map[string]bool
	clearInProgress map[string]bool
}

func NewAdminHandlers(userManager *state.StateManager, adminUsernames []string) *AdminHandlers {
	admins := make(map[string]bool)
	clearInProgress := make(map[string]bool)
	for _, username := range adminUsernames {
		admins[username] = true
		clearInProgress[username] = false
	}

	return &AdminHandlers{
		userManager:     userManager,
		admins:          admins,
		clearInProgress: clearInProgress,
	}
}

func (h *AdminHandlers) clearLineHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message

	if !h.admins[message.From.UserName] {
		return b.SendMessage(message.Chat.ID, "вы не админ")
	}
	h.clearInProgress[update.Message.From.UserName] = true
	return b.SendMessageWithButtons(message.Chat.ID, "весь список будет безвозвратно удалён! уверены?",
		tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("удаляем", "confirm_clear_line"),
				tgbotapi.NewInlineKeyboardButtonData("отмена", "abort_clear_line"),
			),
		),
	)
}

func (h *AdminHandlers) confirmHandler(b *bot.Bot, update tgbotapi.Update) error {
	ctx := context.Background()
	if h.clearInProgress[update.CallbackQuery.From.UserName] {
		h.userManager.Clear(ctx)
		h.clearInProgress[update.CallbackQuery.From.UserName] = false
		return b.SendMessage(update.CallbackQuery.From.ID, "список очищен")
	}
	return b.SendMessage(update.CallbackQuery.From.ID, "кнопка уже не работает")

}

func (h *AdminHandlers) abortHandler(b *bot.Bot, update tgbotapi.Update) error {
	if h.clearInProgress[update.CallbackQuery.From.UserName] {
		h.clearInProgress[update.CallbackQuery.From.UserName] = false
		return b.SendMessage(update.CallbackQuery.From.ID, "ок. отменили")
	}
	return b.SendMessage(update.CallbackQuery.From.ID, "кнопка уже не работает")
}

func SetupHandlers(adminBot *bot.Bot, userManager *state.StateManager, adminUsernames []string) {
	handlers := NewAdminHandlers(userManager, adminUsernames)

	commandHandlers := common.GetCommandHandlers(userManager)
	commandHandlers["clear_line"] = handlers.clearLineHandler

	callbackHandlers := common.GetCallbackHandlers(userManager)
	callbackHandlers["abort_clear_line"] = handlers.abortHandler
	callbackHandlers["confirm_clear_line"] = handlers.confirmHandler

	go adminBot.Start(commandHandlers, nil, callbackHandlers)
}
