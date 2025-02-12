package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
)

func (h *AdminHandlers) RebuildHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	githubWebhookURL := os.Getenv("GITHUB_REDEPLOY_HOOK")
	githubToken := os.Getenv("GITHUB_PAT_TOKEN")

	if githubWebhookURL == "" || githubToken == "" {
		return b.SendMessage(update.Message.Chat.ID, "ошибка: не настроены webhook url или токен")
	}

	payload := map[string]interface{}{
		"event_type": "rebuild-trigger",
		"client_payload": map[string]string{
			"unit": "rebuild triggered via telegram",
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
	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken)) // Changed from "Bearer" to "token"
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return b.SendMessage(update.Message.Chat.ID, fmt.Sprintf("ошибка при запросе к github: %v", err))
	}
	defer resp.Body.Close()

	// Read and log the response body for debugging
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "" {
		fmt.Printf("GitHub Response: %s\n", string(body))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return b.SendMessage(update.Message.Chat.ID,
			fmt.Sprintf("ошибка: получен статус %d от github. ответ: %s",
				resp.StatusCode, string(body)))
	}

	return b.SendMessage(update.Message.Chat.ID, "запущен процесс пересборки сайта")
}
