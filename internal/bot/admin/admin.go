package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/bot/common"
	"github.com/sukalov/karaokebot/internal/db"
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

func (h *AdminHandlers) limitHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	chatID := update.Message.Chat.ID
	limit := h.userManager.GetLimit()
	if limit == 3 {
		return b.SendMessageWithButtons(chatID, "сейчас один человек может спеть максимум 3 песни", tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("убрать лимит", "disable_limit"),
			),
		))
	}
	return b.SendMessageWithButtons(chatID, "сейчас любой может петь сколько угодно раз", tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("включить лимит в три песни", "enable_limit"),
		),
	))
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

func (h *AdminHandlers) openLineHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}
	ctx := context.Background()
	if err := h.userManager.OpenList(ctx); err != nil {
		return b.SendMessage(update.Message.From.ID, "случилась ошибка")
	}
	return b.SendMessage(update.Message.From.ID, "список открыт для записи")
}

func (h *AdminHandlers) closeLineHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}
	ctx := context.Background()
	if err := h.userManager.CloseList(ctx); err != nil {
		return b.SendMessage(update.Message.From.ID, "случилась ошибка")
	}
	return b.SendMessage(update.Message.From.ID, "запись закрыта")
}

func (h *AdminHandlers) enableLimitHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.CallbackQuery.From.UserName] {
		return b.SendMessage(update.CallbackQuery.From.ID, "вы не админ")
	}
	ctx := context.Background()
	h.userManager.SetLimit(ctx, 3)
	return b.SendMessage(update.CallbackQuery.From.ID, "лимит ON. теперь каждый поёт не больше трёх раз")
}

func (h *AdminHandlers) disableLimitHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.CallbackQuery.From.UserName] {
		return b.SendMessage(update.CallbackQuery.From.ID, "вы не админ")
	}
	ctx := context.Background()
	h.userManager.SetLimit(ctx, 1000)
	return b.SendMessage(update.CallbackQuery.From.ID, "лимит OFF. все поют сколько угодно")
}

func (h *AdminHandlers) ShowPromoHandler(b *bot.Bot, update tgbotapi.Update) error {
	return h.updatePromoAndRebuild(b, update, "true")
}

func (h *AdminHandlers) HidePromoHandler(b *bot.Bot, update tgbotapi.Update) error {
	return h.updatePromoAndRebuild(b, update, "false")
}

func (h *AdminHandlers) RebuildHandler(b *bot.Bot, update tgbotapi.Update) error {
	return h.triggerGithubAction(b, update, "rebuild-trigger", "")
}

func (h *AdminHandlers) updatePromoAndRebuild(b *bot.Bot, update tgbotapi.Update, value string) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	githubToken := os.Getenv("GITHUB_PAT_TOKEN")
	repo := "sukalov/karaoke" // update this if needed

	// 1. Update the GitHub Repository Variable
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/actions/variables/NEXT_PUBLIC_SHOW_PROMO", repo)
	payload := map[string]string{"name": "NEXT_PUBLIC_SHOW_PROMO", "value": value}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при подготовке payload: %v", err))
	}

	req, err := http.NewRequest("PATCH", apiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при создании запроса: %v", err))
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || (resp.StatusCode != 204 && resp.StatusCode != 200) {
		// If PATCH fails, it might not exist yet, try POST
		apiUrl = fmt.Sprintf("https://api.github.com/repos/%s/actions/variables", repo)
		jsonPayload, _ = json.Marshal(payload)
		req, _ = http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
		req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
		req.Header.Set("Content-Type", "application/json")
		resp, err = client.Do(req)
	}
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, "ошибка при обновлении переменной в GitHub")
	}
	defer resp.Body.Close()

	// 2. Trigger the rebuild
	rebuildUrl := fmt.Sprintf("https://api.github.com/repos/%s/dispatches", repo)
	rebuildPayload := map[string]interface{}{
		"event_type":     "rebuild-trigger",
		"client_payload": map[string]string{"unit": "rebuild triggered via bot"},
	}
	jsonRebuild, err := json.Marshal(rebuildPayload)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при подготовке rebuild payload: %v", err))
	}

	req, err = http.NewRequest("POST", rebuildUrl, bytes.NewBuffer(jsonRebuild))
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при создании rebuild запроса: %v", err))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, "ошибка при запуске пересборки")
	}
	defer resp.Body.Close()

	return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("состояние обновлено на %s, запущен процесс пересборки", value))
}

func (h *AdminHandlers) triggerGithubAction(b *bot.Bot, update tgbotapi.Update, eventType string, promoValue string) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	githubWebhookURL := os.Getenv("GITHUB_REDEPLOY_HOOK")
	githubToken := os.Getenv("GITHUB_PAT_TOKEN")

	if githubWebhookURL == "" || githubToken == "" {
		return b.SendMessage(update.Message.Chat.ID, "ошибка: не настроены webhook url или токен")
	}

	payload := map[string]interface{}{
		"event_type": eventType,
		"client_payload": map[string]string{
			"unit":  fmt.Sprintf("rebuild triggered via telegram: %s", eventType),
			"promo": promoValue,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при подготовке payload: %v", err))
	}

	req, err := http.NewRequest("POST", githubWebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при создании запроса: %v", err))
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при запросе к github: %v", err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "" {
		fmt.Printf("GitHub Response: %s\n", string(body))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return b.SendMessage(update.Message.Chat.ID,
			fmt.Sprintf("ошибка: получен статус %d от github. ответ: %s",
				resp.StatusCode, string(body)))
	}

	var message string
	switch eventType {
	case "rebuild-trigger":
		message = "запущен процесс пересборки сайта"
	case "show-promo":
		message = "промо-кнопка показана"
	case "hide-promo":
		message = "промо-кнопка скрыта"
	default:
		message = fmt.Sprintf("запущен процесс: %s", eventType)
	}

	return b.SendMessage(update.Message.Chat.ID, message)
}

func SetupHandlers(adminBot *bot.Bot, userManager *state.StateManager, adminUsernames []string) {
	// Create handlers
	handlers := NewAdminHandlers(userManager, adminUsernames)
	songManager := db.Songbook
	searchHandlers := NewSearchHandler(adminUsernames, songManager)

	// Get common handlers
	commandHandlers := common.GetCommandHandlers(userManager)
	messageHandlers := common.GetMessageHandlers()
	callbackHandlers := common.GetCallbackHandlers(userManager)

	// Add admin command handlers
	commandHandlers["clear_line"] = handlers.clearLineHandler
	commandHandlers["rebuild"] = handlers.RebuildHandler
	commandHandlers["show_promo"] = handlers.ShowPromoHandler
	commandHandlers["hide_promo"] = handlers.HidePromoHandler
	commandHandlers["findsong"] = searchHandlers.findSongHandler
	commandHandlers["cancel"] = searchHandlers.cancelAction
	commandHandlers["open"] = handlers.openLineHandler
	commandHandlers["close"] = handlers.closeLineHandler
	commandHandlers["newsong"] = searchHandlers.newSongHandler
	commandHandlers["newsongform"] = searchHandlers.newSongFormHandler
	commandHandlers["limit"] = handlers.limitHandler

	// Add message handler
	messageHandlers = append(messageHandlers, searchHandlers.messageHandler)

	// Add callback handlers for all possible prefixes
	callbackHandlers["edit_song"] = searchHandlers.callbackHandler
	callbackHandlers["edit_field"] = searchHandlers.callbackHandler
	callbackHandlers["delete_song"] = searchHandlers.callbackHandler
	callbackHandlers["select_category"] = searchHandlers.selectCategoryCallbackHandler
	callbackHandlers["abort_clear_line"] = handlers.abortHandler
	callbackHandlers["confirm_clear_line"] = handlers.confirmHandler
	callbackHandlers["enable_limit"] = handlers.enableLimitHandler
	callbackHandlers["disable_limit"] = handlers.disableLimitHandler

	// Start the bot
	go adminBot.Start(commandHandlers, messageHandlers, callbackHandlers)
}
