package admin

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/bot/common"
	"github.com/sukalov/karaokebot/internal/users"
)

type AdminHandlers struct {
	userManager *users.UserStateManager
	admins      map[string]bool
}

func NewAdminHandlers(userManager *users.UserStateManager, adminUsernames []string) *AdminHandlers {
	admins := make(map[string]bool)
	for _, username := range adminUsernames {
		admins[username] = true
	}

	return &AdminHandlers{
		userManager: userManager,
		admins:      admins,
	}
}

func (h *AdminHandlers) clearLineHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message

	if !h.admins[message.From.UserName] {
		return b.SendMessage(message.Chat.ID, "вы не админ")
	}

	h.userManager.Clear()
	return b.SendMessage(message.Chat.ID, "список очищен")
}

func SetupHandlers(adminBot *bot.Bot, userManager *users.UserStateManager, adminUsernames []string) {
	handlers := NewAdminHandlers(userManager, adminUsernames)

	commandHandlers := common.GetCommandHandlers(userManager)
	commandHandlers["clear_line"] = handlers.clearLineHandler

	callbackHandlers := common.GetCallbackHandlers()

	go adminBot.Start(commandHandlers, nil, callbackHandlers)
}
