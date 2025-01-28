package admin

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/db"
)

type SearchHandler struct {
	admins         map[string]bool
	songManager    *db.SongbookType
	awaitingSearch map[int64]bool
}

func NewSearchHandler(adminUsernames []string, songManager *db.SongbookType) *SearchHandler {
	admins := make(map[string]bool)
	for _, username := range adminUsernames {
		admins[username] = true
	}

	return &SearchHandler{
		admins:         admins,
		songManager:    songManager,
		awaitingSearch: make(map[int64]bool),
	}
}

func (h *SearchHandler) findSongHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	h.awaitingSearch[update.Message.Chat.ID] = true
	return b.SendMessage(update.Message.Chat.ID, "здесь можно найти песню и отредактировать. напишите название песни или артиста")
}

func (h *SearchHandler) messageHandler(b *bot.Bot, update tgbotapi.Update) error {
	// Skip if this is a callback query
	if update.CallbackQuery != nil {
		return nil
	}

	if !h.awaitingSearch[update.Message.Chat.ID] {
		return b.SendMessage(update.Message.From.ID, "ничего не понятно. если вы пытаетесь найти песню, сначала нажмите /findsong")
	}

	// Clear the awaiting state
	delete(h.awaitingSearch, update.Message.Chat.ID)

	// Search for songs
	results := h.songManager.SearchSongs(update.Message.Text)

	if len(results) == 0 {
		return b.SendMessage(update.Message.Chat.ID, "ничего не найдено")
	}

	// Create keyboard with search results
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, song := range results {
		// Limit the number of results to prevent huge keyboards
		if len(rows) >= 10 {
			break
		}

		songName := h.songManager.FormatSongName(song)

		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(songName, "edit_song:"+song.ID),
		}
		rows = append(rows, row)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	message := "найденные песни:"
	if len(results) > 10 {
		message += fmt.Sprintf("\n(показаны первые 10 из %d)", len(results))
	}

	return b.SendMessageWithButtons(update.Message.Chat.ID, message, keyboard)
}

func (h *SearchHandler) callbackHandler(b *bot.Bot, update tgbotapi.Update) error {
	fmt.Println(update.CallbackQuery, "ESTTT")
	if update.CallbackQuery == nil {
		return nil
	}

	data := update.CallbackQuery.Data
	if !strings.HasPrefix(data, "edit_song:") {
		return nil
	}

	songID := strings.TrimPrefix(data, "edit_song:")
	song, found := h.songManager.FindSongByID(songID)
	if !found {
		return b.SendMessage(update.CallbackQuery.Message.Chat.ID, "песня не найдена")
	}

	return b.SendMessage(update.CallbackQuery.Message.Chat.ID,
		fmt.Sprintf("редактируем песню:\n%s", h.songManager.FormatSongName(song)))
}
