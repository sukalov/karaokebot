package admin

import (
	"database/sql"
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
	editingSong    map[int64]string // chatID -> songID
	editingField   map[int64]string // chatID -> field name
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
		editingSong:    make(map[int64]string),
		editingField:   make(map[int64]string),
	}
}

func (h *SearchHandler) cancelAction(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}
	chatID := update.Message.Chat.ID
	h.awaitingSearch[chatID] = false
	delete(h.editingSong, chatID)
	delete(h.editingField, chatID)
	return b.SendMessage(chatID, "все действия по сонгбуку завершены")
}

func (h *SearchHandler) findSongHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	h.awaitingSearch[update.Message.Chat.ID] = true
	return b.SendMessage(update.Message.Chat.ID, "здесь можно найти песню и отредактировать. напишите название песни или артиста")
}

func (h *SearchHandler) handleEditSong(b *bot.Bot, chatID int64, songID string) error {
	song, found := h.songManager.FindSongByID(songID)
	if !found {
		return b.SendMessage(chatID, "песня не найдена")
	}

	h.editingSong[chatID] = songID

	var rows [][]tgbotapi.InlineKeyboardButton
	fields := []struct {
		name    string
		display string
		value   string
	}{
		{"category", "категория", song.Category},
		{"title", "название", song.Title},
		{"link", "ссылка", song.Link},
		{"artist", "исполнитель", song.Artist.String},
		{"artistname", "имя исполнителя", song.ArtistName.String},
		{"additionalchords", "доп. аккорды", song.AdditionalChords.String},
		{"excluded", "исключена", h.formatExcluded(song.Excluded)},
	}

	for _, field := range fields {
		currentValue := field.value
		if currentValue == "" {
			currentValue = "не указано"
		}

		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s: %s", field.display, currentValue),
				fmt.Sprintf("edit_field:%s:%s", songID, field.name),
			),
		}
		rows = append(rows, row)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return b.SendMessageWithButtons(chatID,
		fmt.Sprintf("редактирование песни:\n%s\nвыберите поле для редактирования:\n\nчтобы завершить нажмите /cancel",
			h.songManager.FormatSongName(song)),
		keyboard)
}

// func (h *SearchHandler) handleEditField(b *bot.Bot, chatID int64, songID string, field string) error {
// 	_, found := h.songManager.FindSongByID(songID)
// 	if !found {
// 		return b.SendMessage(chatID, "песня не найдена")
// 	}

// 	h.editingSong[chatID] = songID
// 	h.editingField[chatID] = field

// 	var message string
// 	if field == "excluded" {
// 		message = "песня исключена из поиска? (ответьте 'да' или 'нет')"
// 	} else {
// 		message = fmt.Sprintf("введите новое значение для поля '%s':", getFieldDisplayName(field))
// 	}

// 	return b.SendMessage(chatID, message)
// }

func (h *SearchHandler) messageHandler(b *bot.Bot, update tgbotapi.Update) error {
	if update.CallbackQuery != nil {
		return nil
	}

	chatID := update.Message.Chat.ID

	// Handle search functionality
	if h.awaitingSearch[chatID] {
		return h.handleSearch(b, update)
	}

	// Handle field editing
	if songID, editing := h.editingSong[chatID]; editing {
		fmt.Println(h.editingSong[chatID])
		field := h.editingField[chatID]
		if field == "" {
			return nil
		}

		return h.handleFieldUpdate(b, chatID, songID, field, update.Message.Text)
	}

	return b.SendMessage(update.Message.From.ID,
		"ничего не понятно. если вы пытаетесь найти песню, сначала нажмите\n /findsong")
}

func (h *SearchHandler) handleSearch(b *bot.Bot, update tgbotapi.Update) error {
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

func (h *SearchHandler) handleFieldUpdate(b *bot.Bot, chatID int64, songID string, field, value string) error {

	song, found := h.songManager.FindSongByID(songID)
	if !found {
		return b.SendMessage(chatID, "песня не найдена")
	}

	delete(h.editingSong, chatID)
	delete(h.editingField, chatID)

	if field == "excluded" {
		value = strings.ToLower(value)
		if value == "да" {
			song.Excluded = 1
		} else {
			song.Excluded = 0
		}
	} else {
		if err := h.updateSongField(&song, field, value); err != nil {
			return b.SendMessage(chatID, fmt.Sprintf("ошибка: %s", err))
		}
	}

	if err := h.songManager.UpdateSong(song); err != nil {
		fmt.Printf("ошибка при сохранении изменений: %s", err)
		return b.SendMessage(chatID, "ошибка при сохранении изменений")
	}

	return b.SendMessage(chatID,
		fmt.Sprintf("поле успешно обновлено!\nтекущие данные песни:\n%s",
			h.songManager.FormatSongName(song)))
}

func (h *SearchHandler) updateSongField(song *db.Song, field, value string) error {
	switch field {
	case "category":
		song.Category = value
	case "title":
		song.Title = value
	case "link":
		song.Link = value
	case "artist":
		song.Artist = sql.NullString{String: value, Valid: true}
	case "artistname":
		song.ArtistName = sql.NullString{String: value, Valid: true}
	case "additionalchords":
		song.AdditionalChords = sql.NullString{String: value, Valid: true}
	default:
		return fmt.Errorf("неизвестное поле: %s", field)
	}
	return nil
}

func (h *SearchHandler) formatExcluded(excluded int) string {
	if excluded == 1 {
		return "да"
	}
	return "нет"
}

func getFieldDisplayName(field string) string {
	switch field {
	case "category":
		return "категория"
	case "title":
		return "название"
	case "link":
		return "ссылка"
	case "artist":
		return "исполнитель"
	case "artistname":
		return "имя исполнителя"
	case "additionalchords":
		return "дополнительные аккорды"
	case "excluded":
		return "исключена"
	default:
		return field
	}
}

func (h *SearchHandler) callbackHandler(b *bot.Bot, update tgbotapi.Update) error {

	if update.CallbackQuery == nil {
		return nil
	}

	data := update.CallbackQuery.Data
	chatID := update.CallbackQuery.Message.Chat.ID

	fmt.Printf("received callback: %s, chatID: %d\n", data, chatID)

	if strings.HasPrefix(data, "edit_song:") {
		songID := strings.TrimPrefix(data, "edit_song:")
		fmt.Printf("handling edit_song for ID: %s\n", songID)
		if err := h.handleEditSong(b, chatID, songID); err != nil {
			fmt.Printf("error in handleEditSong: %v\n", err)
			return b.SendMessage(chatID, fmt.Sprintf("ошибка: %v", err))
		}
		return nil
	}

	if strings.HasPrefix(data, "edit_field:") {
		fmt.Printf("handling edit_field callback\n")
		parts := strings.Split(strings.TrimPrefix(data, "edit_field:"), ":")
		fmt.Printf("edit_field parts: %v\n", parts)

		if len(parts) != 2 {
			fmt.Printf("invalid parts length: %d\n", len(parts))
			return b.SendMessage(chatID, "ошибка: неверный формат данных")
		}

		songID, field := parts[0], parts[1]
		fmt.Printf("calling handleEditField with songID: %s, field: %s\n", songID, field)

		if err := h.handleEditField(b, chatID, songID, field); err != nil {
			fmt.Printf("error in handleEditField: %v\n", err)
			return b.SendMessage(chatID, fmt.Sprintf("ошибка: %v", err))
		}
		return nil
	}

	fmt.Printf("unhandled callback data: %s\n", data)
	return b.SendMessage(chatID, "неизвестная команда")
}

func (h *SearchHandler) handleEditField(b *bot.Bot, chatID int64, songID string, field string) error {
	fmt.Printf("handleEditField started with songID: %s, field: %s, chatID: %d\n", songID, field, chatID)

	song, found := h.songManager.FindSongByID(songID)
	if !found {
		fmt.Printf("song not found: %s\n", songID)
		return fmt.Errorf("песня не найдена")
	}

	fmt.Printf("song found: %+v\n", song)

	if h.editingSong == nil {
		h.editingSong = make(map[int64]string)
	}
	if h.editingField == nil {
		h.editingField = make(map[int64]string)
	}

	h.editingSong[chatID] = songID
	h.editingField[chatID] = field

	var message string
	if field == "excluded" {
		var value string
		if song.Excluded == 0 {
			value = "да"
		} else {
			value = "нет"
		}
		return h.handleFieldUpdate(b, chatID, songID, field, value)
	} else {
		fieldName := getFieldDisplayName(field)
		currentValue := h.getCurrentFieldValue(song, field)
		message = fmt.Sprintf("продолжаем редактировать '%s'\nсейчас '%s': %s\n\nвведите новое значение:",
			song.Title, fieldName, currentValue)
	}

	fmt.Printf("sending message: %s\n", message)
	err := b.SendMessage(chatID, message)
	if err != nil {
		fmt.Printf("error sending message: %v\n", err)
		return err
	}

	fmt.Printf("handleEditField completed successfully\n")
	return nil
}

// Helper function to get current field value
func (h *SearchHandler) getCurrentFieldValue(song db.Song, field string) string {
	switch field {
	case "category":
		return song.Category
	case "title":
		return song.Title
	case "link":
		return song.Link
	case "artist":
		if song.Artist.Valid {
			return song.Artist.String
		}
		return "не указано"
	case "artistname":
		if song.ArtistName.Valid {
			return song.ArtistName.String
		}
		return "не указано"
	case "additionalchords":
		if song.AdditionalChords.Valid {
			return song.AdditionalChords.String
		}
		return "не указано"
	case "excluded":
		if song.Excluded == 1 {
			return "да"
		}
		return "нет"
	default:
		return "неизвестное поле"
	}
}
