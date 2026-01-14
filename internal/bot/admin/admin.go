package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/bot/common"
	"github.com/sukalov/karaokebot/internal/db"
	"github.com/sukalov/karaokebot/internal/logger"
	"github.com/sukalov/karaokebot/internal/lyrics"
	"github.com/sukalov/karaokebot/internal/state"
)

type PromoEditState struct {
	currentText string
	currentURL  string
	editingText bool
	editingURL  bool
	messageID   int // Track the message to edit
}

type AdminHandlers struct {
	userManager     *state.StateManager
	admins          map[string]bool
	clearInProgress map[string]bool
	promoEditState  map[int64]*PromoEditState
	lyricsService   *lyrics.Service
}

func NewAdminHandlers(userManager *state.StateManager, adminUsernames []string) *AdminHandlers {
	admins := make(map[string]bool)
	clearInProgress := make(map[string]bool)
	promoEditState := make(map[int64]*PromoEditState)
	for _, username := range adminUsernames {
		admins[username] = true
		clearInProgress[username] = false
	}

	return &AdminHandlers{
		userManager:     userManager,
		admins:          admins,
		clearInProgress: clearInProgress,
		promoEditState:  promoEditState,
		lyricsService:   lyrics.NewService(),
	}
}

func (h *AdminHandlers) clearLineHandler(b *bot.Bot, update tgbotapi.Update) error {
	message := update.Message

	if !h.admins[message.From.UserName] {
		return b.SendMessage(message.Chat.ID, "вы не админ")
	}
	h.clearInProgress[update.Message.From.UserName] = true
	logger.Info(fmt.Sprintf("⚙️📋 [INFO] Admin %s initiated clear line", message.From.UserName))
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

func (h *AdminHandlers) testLyricsHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	message := update.Message
	text := message.Text

	// Extract URL from command
	args := strings.TrimPrefix(text, "/test_lyrics ")
	if args == text {
		return b.SendMessage(message.Chat.ID, "использование: /test_lyrics <URL>")
	}

	url := strings.TrimSpace(args)
	logger.Info(fmt.Sprintf("⚙️📋 [INFO] Admin %s requested lyrics test", message.From.UserName))

	if !strings.Contains(url, "amdm.ru") {
		return b.SendMessage(message.Chat.ID, "поддерживаются только ссылки с amdm.ru")
	}

	b.SendMessage(message.Chat.ID, "загружаю слова...")

	result, err := h.lyricsService.ExtractLyrics(url)
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to extract lyrics for test\nAdmin: %s\nError: %v", message.From.UserName, err))
		return b.SendMessage(message.Chat.ID, fmt.Sprintf("ошибка при извлечении слов: %v", err))
	}

	logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Lyrics test succeeded for admin %s\nLength: %d chars", message.From.UserName, len(result.Text)))

	if len(result.Text) > 3000 {
		truncated := result.Text[:3000] + "\n\n... (обрезано, текст слишком длинный)"
		return b.SendMessageWithMarkdown(message.Chat.ID, truncated, false)
	}

	return b.SendMessageWithMarkdown(message.Chat.ID, result.Text, false)
}

func (h *AdminHandlers) confirmHandler(b *bot.Bot, update tgbotapi.Update) error {
	ctx := context.Background()
	if h.clearInProgress[update.CallbackQuery.From.UserName] {
		h.userManager.Clear(ctx)
		h.clearInProgress[update.CallbackQuery.From.UserName] = false
		logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Admin %s cleared the line", update.CallbackQuery.From.UserName))
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
	logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Admin %s opened line", update.Message.From.UserName))
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
	logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Admin %s closed line", update.Message.From.UserName))
	return b.SendMessage(update.Message.From.ID, "запись закрыта")
}

func (h *AdminHandlers) enableLimitHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.CallbackQuery.From.UserName] {
		return b.SendMessage(update.CallbackQuery.From.ID, "вы не админ")
	}
	ctx := context.Background()
	h.userManager.SetLimit(ctx, 3)
	logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Admin %s enabled limit (3 songs)", update.CallbackQuery.From.UserName))
	return b.SendMessage(update.CallbackQuery.From.ID, "лимит ON. теперь каждый поёт не больше трёх раз")
}

func (h *AdminHandlers) disableLimitHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.CallbackQuery.From.UserName] {
		return b.SendMessage(update.CallbackQuery.From.ID, "вы не админ")
	}
	ctx := context.Background()
	h.userManager.SetLimit(ctx, 1000)
	logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Admin %s disabled limit", update.CallbackQuery.From.UserName))
	return b.SendMessage(update.CallbackQuery.From.ID, "лимит OFF. все поют сколько угодно")
}

func (h *AdminHandlers) ShowPromoHandler(b *bot.Bot, update tgbotapi.Update) error {
	return h.updatePromoAndRebuild(b, update, "true")
}

func (h *AdminHandlers) HidePromoHandler(b *bot.Bot, update tgbotapi.Update) error {
	return h.updatePromoAndRebuild(b, update, "false")
}

func (h *AdminHandlers) EditPromoHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	chatID := update.Message.Chat.ID

	// Fetch current promo values
	currentText, currentURL, err := h.fetchCurrentPromoValues()
	if err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("ошибка при получении текущих значений: %v", err))
	}

	// Store editing state
	h.promoEditState[chatID] = &PromoEditState{
		currentText: currentText,
		currentURL:  currentURL,
		editingText: false,
		editingURL:  false,
	}

	return h.sendPromoEditMessage(b, chatID, 0)
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
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to marshal GitHub variable payload\nError: %v", err))
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при подготовке payload: %v", err))
	}

	req, err := http.NewRequest("PATCH", apiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to create GitHub PATCH request\nError: %v", err))
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
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to update GitHub variable NEXT_PUBLIC_SHOW_PROMO\nError: %v", err))
		return b.SendMessage(update.Message.Chat.ID, "ошибка при обновлении переменной в GitHub")
	}
	defer resp.Body.Close()

	// 2. Wait for variable propagation
	time.Sleep(5 * time.Second)

	// 3. Trigger the rebuild with the value in payload to avoid race condition
	rebuildUrl := fmt.Sprintf("https://api.github.com/repos/%s/dispatches", repo)
	rebuildPayload := map[string]interface{}{
		"event_type": "rebuild-trigger",
		"client_payload": map[string]interface{}{
			"unit":       "rebuild triggered via bot",
			"promo_show": value,
		},
	}
	jsonRebuild, err := json.Marshal(rebuildPayload)
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to marshal rebuild payload\nError: %v", err))
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при подготовке rebuild payload: %v", err))
	}

	req, err = http.NewRequest("POST", rebuildUrl, bytes.NewBuffer(jsonRebuild))
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to create rebuild request\nError: %v", err))
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при создании rebuild запроса: %v", err))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to trigger rebuild\nError: %v", err))
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
		"client_payload": map[string]interface{}{
			"unit":  fmt.Sprintf("rebuild triggered via telegram: %s", eventType),
			"promo": promoValue,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to marshal GitHub webhook payload\nError: %v", err))
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при подготовке payload: %v", err))
	}

	req, err := http.NewRequest("POST", githubWebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to create GitHub webhook request\nError: %v", err))
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при создании запроса: %v", err))
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to send GitHub webhook request\nError: %v", err))
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
	var logMessage string
	switch eventType {
	case "rebuild-trigger":
		message = "запущен процесс пересборки сайта"
		logMessage = fmt.Sprintf("⚙️✅ [SUCCESS] Admin %s triggered rebuild", update.Message.From.UserName)
	case "show-promo":
		message = "промо-кнопка показана"
	case "hide-promo":
		message = "промо-кнопка скрыта"
	default:
		message = fmt.Sprintf("запущен процесс: %s", eventType)
	}
	if logMessage != "" {
		logger.Success(logMessage)
	}

	return b.SendMessage(update.Message.Chat.ID, message)
}

func (h *AdminHandlers) fetchCurrentPromoValues() (string, string, error) {
	githubToken := os.Getenv("GITHUB_PAT_TOKEN")
	repo := "sukalov/karaoke"

	// Default values if variables don't exist
	defaultText := "🔥 мы продаёмся 🔥"
	defaultURL := "https://karaoke.gastroli.moscow"

	// Fetch NEXT_PUBLIC_PROMO_TEXT
	textURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/variables/NEXT_PUBLIC_PROMO_TEXT", repo)
	text, err := h.fetchGitHubVariable(textURL, githubToken)
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to fetch NEXT_PUBLIC_PROMO_TEXT from GitHub\nError: %v", err))
		text = defaultText
	}

	// Fetch NEXT_PUBLIC_PROMO_URL
	urlURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/variables/NEXT_PUBLIC_PROMO_URL", repo)
	promoURL, err := h.fetchGitHubVariable(urlURL, githubToken)
	if err != nil {
		logger.Error(fmt.Sprintf("⚙️🔴 [ERROR] Failed to fetch NEXT_PUBLIC_PROMO_URL from GitHub\nError: %v", err))
		promoURL = defaultURL
	}

	return text, promoURL, nil
}

func (h *AdminHandlers) fetchGitHubVariable(variableURL, token string) (string, error) {
	req, err := http.NewRequest("GET", variableURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response struct {
		Value string `json:"value"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	return response.Value, nil
}

func (h *AdminHandlers) sendPromoEditMessage(b *bot.Bot, chatID int64, messageID int) error {
	state, exists := h.promoEditState[chatID]
	if !exists {
		return fmt.Errorf("editing state not found")
	}

	var message string
	if state.editingText {
		message = "введите новый текст для промо-кнопки:"
	} else if state.editingURL {
		message = "введите новый URL для промо-кнопки:"
	} else {
		message = fmt.Sprintf("текущие настройки промо:\nтекст: \"%s\"\nURL: %s", state.currentText, state.currentURL)
	}

	buttons := tgbotapi.NewInlineKeyboardMarkup()

	if state.editingText {
		buttons = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ отмена", "cancel_promo_edit"),
			),
		)
	} else if state.editingURL {
		buttons = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ отмена", "cancel_promo_edit"),
			),
		)
	} else {
		buttons = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✏️ изменить текст", "edit_promo_text"),
				tgbotapi.NewInlineKeyboardButtonData("🔗 изменить URL", "edit_promo_url"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("✅ подтвердить", "confirm_promo_edit"),
			),
		)
	}

	if messageID == 0 {
		msg := tgbotapi.NewMessage(chatID, message)
		msg.ReplyMarkup = buttons
		msg.ParseMode = "Markdown"
		msg.DisableWebPagePreview = true

		sentMsg, err := b.Client.Send(msg)
		if err != nil {
			return err
		}

		state.messageID = sentMsg.MessageID
	} else {
		editMsg := tgbotapi.NewEditMessageText(chatID, messageID, message)
		editMsg.ReplyMarkup = &buttons
		editMsg.ParseMode = "Markdown"

		_, err := b.Client.Send(editMsg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *AdminHandlers) editPromoCallbackHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.CallbackQuery.From.UserName] {
		return b.SendMessage(update.CallbackQuery.Message.Chat.ID, "вы не админ")
	}

	chatID := update.CallbackQuery.Message.Chat.ID
	data := update.CallbackQuery.Data

	state, exists := h.promoEditState[chatID]
	if !exists {
		return b.SendMessage(chatID, "сессия редактирования не найдена")
	}

	switch data {
	case "edit_promo_text":
		state.editingText = true
		state.editingURL = false
		return h.sendPromoEditMessage(b, chatID, state.messageID)

	case "edit_promo_url":
		state.editingText = false
		state.editingURL = true
		return h.sendPromoEditMessage(b, chatID, state.messageID)

	case "confirm_promo_edit":
		return h.updatePromoVariablesAndRebuild(b, chatID, state.currentText, state.currentURL)

	case "cancel_promo_edit":
		delete(h.promoEditState, chatID)
		return b.SendMessage(chatID, "редактирование промо отменено")
	}

	return nil
}

func (h *AdminHandlers) handlePromoMessageInput(b *bot.Bot, update tgbotapi.Update) error {
	chatID := update.Message.Chat.ID
	text := update.Message.Text

	state, exists := h.promoEditState[chatID]
	if !exists {
		return nil // Let search handler process normal messages
	}

	if state.editingText {
		if strings.TrimSpace(text) == "" {
			return b.SendMessage(chatID, "текст не может быть пустым")
		}
		state.currentText = strings.TrimSpace(text)
		state.editingText = false
		err := h.sendPromoEditMessage(b, chatID, state.messageID)
		if err == nil {
			return bot.ErrMessageHandled
		}
		return err
	}

	if state.editingURL {
		if strings.TrimSpace(text) == "" {
			return b.SendMessage(chatID, "URL не может быть пустым")
		}

		// Basic URL validation
		if !strings.HasPrefix(text, "http://") && !strings.HasPrefix(text, "https://") {
			return b.SendMessage(chatID, "URL должен начинаться с http:// или https://")
		}

		if _, err := url.Parse(text); err != nil {
			return b.SendMessage(chatID, "неверный формат URL")
		}

		state.currentURL = strings.TrimSpace(text)
		state.editingURL = false
		err := h.sendPromoEditMessage(b, chatID, state.messageID)
		if err == nil {
			return bot.ErrMessageHandled
		}
		return err
	}

	return nil
}

func (h *AdminHandlers) updatePromoVariablesAndRebuild(b *bot.Bot, chatID int64, text, promoURL string) error {
	githubToken := os.Getenv("GITHUB_PAT_TOKEN")
	repo := "sukalov/karaoke"

	// Update NEXT_PUBLIC_PROMO_TEXT
	if err := h.updateGitHubVariable(repo, githubToken, "NEXT_PUBLIC_PROMO_TEXT", text); err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("ошибка при обновлении текста: %v", err))
	}

	// Update NEXT_PUBLIC_PROMO_URL
	if err := h.updateGitHubVariable(repo, githubToken, "NEXT_PUBLIC_PROMO_URL", promoURL); err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("ошибка при обновлении URL: %v", err))
	}

	// Wait a moment for repository variable propagation
	time.Sleep(5 * time.Second)

	// Trigger rebuild with new values in payload to avoid race condition
	rebuildUrl := fmt.Sprintf("https://api.github.com/repos/%s/dispatches", repo)
	rebuildPayload := map[string]interface{}{
		"event_type": "rebuild-trigger",
		"client_payload": map[string]interface{}{
			"unit":       "rebuild triggered via promo edit",
			"promo_text": text,
			"promo_url":  promoURL,
		},
	}
	jsonRebuild, err := json.Marshal(rebuildPayload)
	if err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("ошибка при подготовке rebuild payload: %v", err))
	}

	req, err := http.NewRequest("POST", rebuildUrl, bytes.NewBuffer(jsonRebuild))
	if err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("ошибка при создании rebuild запроса: %v", err))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return b.SendMessage(chatID, "ошибка при запуске пересборки")
	}
	defer resp.Body.Close()

	// Clean up state
	delete(h.promoEditState, chatID)

	confirmation := fmt.Sprintf("промо обновлено!\nтекст: \"%s\"\nURL: %s\n\nзапущен процесс пересборки сайта", text, promoURL)
	return b.SendMessage(chatID, confirmation)
}

func (h *AdminHandlers) updateGitHubVariable(repo, token, name, value string) error {
	apiUrl := fmt.Sprintf("https://api.github.com/repos/%s/actions/variables/%s", repo, name)
	payload := map[string]string{"name": name, "value": value}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PATCH", apiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err == nil && (resp.StatusCode == 204 || resp.StatusCode == 200) {
		resp.Body.Close()
		return nil
	}
	if resp != nil {
		resp.Body.Close()
	}

	// PATCH failed, try POST to create the variable
	apiUrl = fmt.Sprintf("https://api.github.com/repos/%s/actions/variables", repo)
	req, err = http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("failed to create variable: %s", resp.Status)
	}

	return nil
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
	commandHandlers["edit_promo"] = handlers.EditPromoHandler
	commandHandlers["findsong"] = searchHandlers.findSongHandler
	commandHandlers["cancel"] = searchHandlers.cancelAction
	commandHandlers["open"] = handlers.openLineHandler
	commandHandlers["close"] = handlers.closeLineHandler
	commandHandlers["newsong"] = searchHandlers.newSongHandler
	commandHandlers["newsongform"] = searchHandlers.newSongFormHandler
	commandHandlers["limit"] = handlers.limitHandler
	commandHandlers["test_lyrics"] = handlers.testLyricsHandler

	// Add message handler
	messageHandlers = append(messageHandlers, handlers.handlePromoMessageInput, searchHandlers.messageHandler)

	// Add callback handlers for all possible prefixes
	callbackHandlers["edit_song"] = searchHandlers.callbackHandler
	callbackHandlers["edit_field"] = searchHandlers.callbackHandler
	callbackHandlers["delete_song"] = searchHandlers.callbackHandler
	callbackHandlers["select_category"] = searchHandlers.selectCategoryCallbackHandler
	callbackHandlers["abort_clear_line"] = handlers.abortHandler
	callbackHandlers["confirm_clear_line"] = handlers.confirmHandler
	callbackHandlers["enable_limit"] = handlers.enableLimitHandler
	callbackHandlers["disable_limit"] = handlers.disableLimitHandler
	callbackHandlers["edit_promo_text"] = handlers.editPromoCallbackHandler
	callbackHandlers["edit_promo_url"] = handlers.editPromoCallbackHandler
	callbackHandlers["confirm_promo_edit"] = handlers.editPromoCallbackHandler
	callbackHandlers["cancel_promo_edit"] = handlers.editPromoCallbackHandler

	// Start the bot
	go adminBot.Start(commandHandlers, messageHandlers, callbackHandlers)
}
