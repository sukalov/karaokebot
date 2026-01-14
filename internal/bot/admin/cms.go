// Package admin provides functionality for managing karaoke bot administrative operations.
package admin

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/db"
	"github.com/sukalov/karaokebot/internal/logger"
)

type SearchHandler struct {
	admins         map[string]bool
	songManager    *db.SongbookType
	awaitingSearch map[int64]bool
	editingSong    map[int64]string  // chatID -> songID
	editingField   map[int64]string  // chatID -> field name
	addingSong     map[int64]db.Song // chatID -> song
	mu             sync.RWMutex
}

var categories = map[string]string{
	"russian_rock": "русский рок",
	"soviet":       "советское",
	"foreign":      "зарубежное",
	"for_kids":     "детские песни",
	"russian_pop":  "русская поп-музыка",
	"different":    "разное",
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
		addingSong:     make(map[int64]db.Song),
		mu:             sync.RWMutex{},
	}
}

func (h *SearchHandler) cancelAction(b *bot.Bot, update tgbotapi.Update) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}
	chatID := update.Message.Chat.ID
	h.awaitingSearch[chatID] = false
	delete(h.editingSong, chatID)
	delete(h.editingField, chatID)
	delete(h.addingSong, chatID)
	return b.SendMessage(chatID, "все действия по сонгбуку завершены")
}

func (h *SearchHandler) findSongHandler(b *bot.Bot, update tgbotapi.Update) error {
	h.mu.Lock()
	if !h.admins[update.Message.From.UserName] {
		h.mu.Unlock()
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	h.awaitingSearch[update.Message.Chat.ID] = true
	h.mu.Unlock()
	logger.Info(fmt.Sprintf("⚙️📋 [INFO] Admin %s initiated song search", update.Message.From.UserName))
	return b.SendMessage(update.Message.Chat.ID, "здесь можно найти песню и отредактировать. напишите название песни или артиста")
}

func (h *SearchHandler) handleEditSong(b *bot.Bot, chatID int64, songID string) error {
	song, found := h.songManager.FindSongByID(songID)
	if !found {
		return b.SendMessage(chatID, "песня не найдена")
	}

	h.editingSong[chatID] = songID
	logger.Info(fmt.Sprintf("⚙️📋 [INFO] Editing song: %s", songID))

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
		{"additionalchords", "заметка:", song.AdditionalChords.String},
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
	deleteButton := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintln("УДАЛИТЬ ИЗ СОНГБУКА"),
			fmt.Sprintf("delete_song:%s", songID),
		),
	}
	rows = append(rows, deleteButton)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return b.SendMessageWithButtons(chatID,
		fmt.Sprintf("редактирование песни:\n[%s](%s)\nвыберите поле для редактирования:\n\nчтобы завершить нажмите /cancel\n\nчтобы найти новую песню нажмите /findsong",
			h.songManager.FormatSongName(song), song.Link),
		keyboard)
}

func (h *SearchHandler) messageHandler(b *bot.Bot, update tgbotapi.Update) error {
	if update.CallbackQuery != nil {
		return nil
	}

	chatID := update.Message.Chat.ID

	if h.awaitingSearch[chatID] {
		return h.handleSearch(b, update)
	}

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
	delete(h.awaitingSearch, update.Message.Chat.ID)

	results := h.songManager.SearchSongs(update.Message.Text)

	if len(results) == 0 {
		return b.SendMessage(update.Message.Chat.ID, "ничего не найдено")
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, song := range results {
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

	logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Updated field '%s' for song: %s", field, songID))
	return b.SendMessage(chatID,
		fmt.Sprintf("поле успешно обновлено!\nтекущие данные песни:\n%s",
			song.Stringify(false)))
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
		return "заметка"
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

	if strings.HasPrefix(data, "edit_song:") {
		songID := strings.TrimPrefix(data, "edit_song:")
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

	if strings.HasPrefix(data, "delete_song:") {
		songID := strings.TrimPrefix(data, "delete_song:")
		if err := h.handleDeleteSong(b, chatID, songID); err != nil {
			fmt.Printf("error in handleDeleteSong: %v\n", err)
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

func (h *SearchHandler) handleDeleteSong(b *bot.Bot, chatID int64, songID string) error {
	_, found := h.songManager.FindSongByID(songID)
	if !found {
		return fmt.Errorf("песня не найдена")
	}

	if err := h.songManager.DeleteSong(songID); err != nil {
		return err
	}

	logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Deleted song: %s", songID))
	return b.SendMessage(chatID, "песня удалена")
}

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

func (h *SearchHandler) newSongHandler(b *bot.Bot, update tgbotapi.Update) error {
	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(update.Message.Chat.ID, "вы не админ")
	}

	chatID := update.Message.Chat.ID
	logger.Info(fmt.Sprintf("⚙️📋 [INFO] Admin %s initiated adding new song", update.Message.From.UserName))
	if err := b.SendMessageWithMarkdown(chatID, "*скопируйте* следующее вообщение (отдним кликом по тексту) и вставьте в него данные новой песни ровно *внутрь квадрятных скобок*. не убирайте квадратные скобки, редактируйте только внтури них, звёздочкой помечены обязательные поля.\n\nп.с. в графе \"исполнитель\" пишется либо название группы либо фамилия исполнителя.", true); err != nil {
		return err
	}
	return b.SendMessageWithMarkdown(chatID, "`/newsongform\n\nисполнитель - []*\nимя исполниеля - []\nназвание песни - []*\nссылка на аккорды - []*`", false)
}

func (h *SearchHandler) newSongFormHandler(b *bot.Bot, update tgbotapi.Update) error {
	chatID := update.Message.Chat.ID

	if !h.admins[update.Message.From.UserName] {
		return b.SendMessage(chatID, "вы не админ")
	}

	song, err := parseNewSongForm(update.Message.Text)
	if err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("ошибка при обработке формы: %v", err))
	}

	if err := h.requestCategoryForNewSong(b, chatID, song); err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("ошибка при отправке сообщения для выбора категории: %v", err))
	}

	return nil
}

func (h *SearchHandler) requestCategoryForNewSong(b *bot.Bot, chatID int64, song *db.Song) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	savedSong := h.addingSong[chatID]
	if savedSong.ID != "" {
		delete(h.addingSong, chatID)
	}

	h.addingSong[chatID] = *song

	var rows [][]tgbotapi.InlineKeyboardButton
	categories := []struct {
		name    string
		display string
	}{
		{"russian_rock", "русский рок"},
		{"soviet", "советское"},
		{"foreign", "зарубежное"},
		{"for_kids", "детские песни"},
		{"russian_pop", "русская поп-музыка"},
		{"different", "разное"}}

	for _, category := range categories {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprint(category.display),
				fmt.Sprintf("select_category:%s", category.name),
			),
		}
		rows = append(rows, row)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	return b.SendMessageWithButtons(chatID,
		fmt.Sprintf("%s\n\nосталось определить её в какую-нибудь категорию\n\nчтобы завершить нажмите /cancel",
			song.Stringify(true)),
		keyboard)
}

func parseNewSongForm(message string) (*db.Song, error) {
	if !strings.HasPrefix(message, "/newsongform") {
		return nil, fmt.Errorf("message should start with /newsongform")
	}

	song := &db.Song{
		ID:        generateRandomID(),
		Counter:   0,
		Excluded:  0,
		CreatedAt: time.Now().Unix(),
	}

	lines := strings.Split(message, "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("incomplete form")
	}

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		dashIndex := strings.Index(line, "-")
		if dashIndex <= 0 {
			continue
		}

		field := strings.TrimSpace(line[:dashIndex])
		valueWithBrackets := strings.TrimSpace(line[dashIndex+1:])

		startBracket := strings.Index(valueWithBrackets, "[")
		endBracket := strings.LastIndex(valueWithBrackets, "]")

		if startBracket == -1 || endBracket == -1 || startBracket >= endBracket {
			continue
		}

		value := strings.TrimSpace(valueWithBrackets[startBracket+1 : endBracket])
		value = strings.TrimRight(value, "*\\")

		switch field {
		case "исполнитель":
			song.Artist = sql.NullString{
				String: value,
				Valid:  value != "",
			}
		case "имя исполниеля":
			song.ArtistName = sql.NullString{
				String: value,
				Valid:  value != "",
			}
		case "название песни":
			if value == "" {
				return nil, fmt.Errorf("должно быть название песни")
			}
			song.Title = value
		case "ссылка на аккорды":
			if value == "" {
				return nil, fmt.Errorf("должна быть ссылка на аккорды")
			}
			song.Link = value
		default:
			return nil, fmt.Errorf("неизвестное поле: %s", field)
		}
	}

	if song.Title == "" || song.Link == "" {
		return nil, fmt.Errorf("required fields are missing")
	}

	return song, nil
}

func generateRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func (h *SearchHandler) selectCategoryCallbackHandler(b *bot.Bot, update tgbotapi.Update) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if update.CallbackQuery == nil {
		return nil
	}

	data := update.CallbackQuery.Data
	chatID := update.CallbackQuery.Message.Chat.ID

	fmt.Printf("received callback: %s, chatID: %d\n", data, chatID)

	if strings.HasPrefix(data, "select_category:") {
		category := strings.TrimPrefix(data, "select_category:")
		if err := h.handleSelectCategory(b, chatID, category); err != nil {
			fmt.Printf("error in handleEditSong: %v\n", err)
			return fmt.Errorf("ошибка: %v", err)
		}
		return nil
	}

	fmt.Printf("unhandled callback data: %s\n", data)
	return b.SendMessage(chatID, "неизвестная команда")
}

func (h *SearchHandler) handleSelectCategory(b *bot.Bot, chatID int64, category string) error {
	song := h.addingSong[chatID]
	if song.ID == "" {
		return b.SendMessage(chatID, "эта кнопка уже не работает. добавьте песню заново")
	}
	song.Category = categories[category]

	if !db.Songbook.ValidateCategory(song.Category) {
		return b.SendMessage(chatID, "неверная категория")
	}

	if err := h.songManager.NewSong(song); err != nil {
		return b.SendMessage(chatID, fmt.Sprintf("ошибка при добавлении песни: %v", err))
	}
	delete(h.addingSong, chatID)
	logger.Success(fmt.Sprintf("⚙️✅ [SUCCESS] Added new song: %s (%s)", song.Title, song.ID))
	return b.SendMessage(chatID, fmt.Sprintf("песня добавлена \n\n%s\n\nне забудьте после всех изменений нажать /rebuild чтобы они появились на сайте", song.Stringify(false)))
}
