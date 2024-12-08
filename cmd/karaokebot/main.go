package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/songbook"
	"github.com/sukalov/karaokebot/internal/utils"
)

var (
	// adminBotToken string
	songs      []songbook.Song
	userStates []UserState
	// admins        = []string{"sukalov", "olakotr", "yatskovanatoly", "motygapishet"}
	userStatesMux sync.RWMutex
)

type UserState struct {
	Username  string    `json:"username"`
	TgName    string    `json:"tg_name"`
	SongID    string    `json:"song_id"`
	SongName  string    `json:"song_name"`
	SongLink  string    `json:"song_link"`
	TypedName string    `json:"typed_name"`
	TimeAdded time.Time `json:"time_added"`
	Stage     string    `json:"stage"`
}

type Tokens struct {
	clientBotToken string
	adminBotToken  string
}

func handleUsers(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userStatesMux.RLock()
	defer userStatesMux.RUnlock()

	// Convert times to Moscow time for JSON output
	var processedStates []UserState
	for _, state := range userStates {
		processedState := state
		processedState.TimeAdded = state.TimeAdded.In(time.FixedZone("Europe/Moscow", 3*60*60))
		processedStates = append(processedStates, processedState)
	}

	statesJSON, err := json.MarshalIndent(processedStates, "", "    ")
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "произошла ошибка при обработке")
		bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("```json\n%s\n```", string(statesJSON)))
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func handleLine(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userStatesMux.RLock()
	defer userStatesMux.RUnlock()

	var lineUsers []UserState
	for _, entry := range userStates {
		if entry.Stage == "in_line" {
			lineUsers = append(lineUsers, entry)
		}
	}

	if len(lineUsers) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "в очереди никого нет")
		bot.Send(msg)
		return
	}

	var lineMessage strings.Builder
	lineMessage.WriteString("очередь:\n\n")
	for idx, user := range lineUsers {
		lineMessage.WriteString(fmt.Sprintf(
			"%d. %s\n"+
				"   песня: [%s](%s)\n"+
				"   добавлен: %s\n"+
				"   юзернейм: @%s\n\n",
			idx+1,
			user.TypedName,
			user.SongName,
			user.SongLink,
			utils.ConvertToMoscowTime(user.TimeAdded),
			user.Username,
		))
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, lineMessage.String())
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true
	bot.Send(msg)
}

func handleExit(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userStatesMux.Lock()
	defer userStatesMux.Unlock()

	var userEntries []int
	for i, entry := range userStates {
		if entry.Username == update.Message.From.UserName && entry.Stage == "in_line" {
			userEntries = append(userEntries, i)
		}
	}

	if len(userEntries) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "вас нет ни в одной очереди")
		bot.Send(msg)
		return
	}

	if len(userEntries) > 1 {
		// Create inline keyboard for multiple entries
		var keyboard [][]tgbotapi.InlineKeyboardButton
		for _, idx := range userEntries {
			entry := userStates[idx]
			button := tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%s - %s", entry.TypedName, entry.SongName),
				fmt.Sprintf("exit_song_%d", idx),
			)
			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "у вас несколько песен в очереди. выберите, от какой хотите выйти:")
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
		bot.Send(msg)
		return
	}

	// Single entry case
	removeUserFromLine(bot, update.Message.Chat.ID, userEntries[0])
}

func removeUserFromLine(bot *tgbotapi.BotAPI, chatID int64, listIdx int) {
	userStatesMux.Lock()
	defer userStatesMux.Unlock()

	if listIdx < 0 || listIdx >= len(userStates) {
		return
	}

	removedEntry := userStates[listIdx]
	userStates = append(userStates[:listIdx], userStates[listIdx+1:]...)

	// Send confirmation
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("вы вышли из очереди. песня: %s", removedEntry.SongName))
	bot.Send(msg)

	// Notify admin (simplified, you might want to use another bot instance)
	adminMsg := tgbotapi.NewMessage(289745814,
		fmt.Sprintf("%s вышел из очереди. песня: %s",
			removedEntry.TypedName,
			removedEntry.SongName))
	bot.Send(adminMsg)
}

func handleStart(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userStatesMux.Lock()
	defer userStatesMux.Unlock()

	parts := strings.Split(update.Message.Text, "/start ")
	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "не, просто так не работает. выбор песен в сонгбуке: https://karaoke-songbook.netlify.app")
		bot.Send(msg)
		return
	}

	songID := parts[1]
	song, found := songbook.FindSongByID(songID)
	if !found {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "извините, песня с таким id не найдена")
		bot.Send(msg)
		return
	}

	// Check user's existing entries
	var userEntries []UserState
	for _, entry := range userStates {
		if entry.Username == update.Message.From.UserName && entry.Stage == "in_line" {
			userEntries = append(userEntries, entry)
		}
	}

	if len(userEntries) >= 3 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "вы уже записаны на 3 песни")
		bot.Send(msg)
		return
	}

	// Create new user state entry
	newUserState := UserState{
		Username: update.Message.From.UserName,
		TgName:   fmt.Sprintf("%s %s", update.Message.From.FirstName, update.Message.From.LastName),
		SongID:   songID,
		SongName: songbook.FormatSongName(song),
		SongLink: song.Link,
		Stage:    "asking_name",
	}
	userStates = append(userStates, newUserState)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("привет! выбрана песня \"%s\". чтобы записаться, скажите, как вас зовут? (или того кто будет петь)\n\n"+
			"если вы передумали — можете выписаться командой /exit", newUserState.SongName))
	bot.Send(msg)
}

func handleNameInput(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	userStatesMux.Lock()
	defer userStatesMux.Unlock()

	for i, entry := range userStates {
		if entry.Username == update.Message.From.UserName && entry.Stage == "asking_name" {
			userStates[i].TypedName = update.Message.Text
			userStates[i].TimeAdded = time.Now()
			userStates[i].Stage = "in_line"

			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				fmt.Sprintf("отлично, %s! вы выбрали песню \"%s\". скоро вас позовут на сцену\n\nа слова можно найти [здесь](%s)",
					userStates[i].TypedName,
					userStates[i].SongName,
					userStates[i].SongLink))
			msg.ParseMode = "Markdown"
			bot.Send(msg)
			return
		}
	}
}

func main() {
	tokens := mustTokens()

	bot, err := tgbotapi.NewBotAPI(tokens.clientBotToken)
	if err != nil {
		log.Panic(err)
	}

	// bot.Debug = true
	log.Printf("authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// Handle inline callback queries
		if update.CallbackQuery != nil {
			continue // TODO: Implement callback handler
		}

		if update.Message == nil {
			continue
		}

		// Skip processing for non-text messages
		if update.Message.Text == "" {
			continue
		}

		// Command handlers
		switch {
		case update.Message.Text == "/users":
			handleUsers(bot, update)
		case update.Message.Text == "/line":
			handleLine(bot, update)
		case update.Message.Text == "/exit":
			handleExit(bot, update)
		case strings.HasPrefix(update.Message.Text, "/start"):
			handleStart(bot, update)
		default:
			// Check if this is a name input
			for _, entry := range userStates {
				if entry.Username == update.Message.From.UserName && entry.Stage == "asking_name" {
					handleNameInput(bot, update)
					break
				}
			}
		}
	}
}

func mustTokens() Tokens {
	tokens, err := utils.LoadEnv([]string{"BOT_TOKEN", "ADMIN_BOT_TOKEN"})

	if err != nil {
		log.Fatal("required env missing")
	}

	return Tokens{adminBotToken: tokens["ADMIN_BOT_TOKEN"], clientBotToken: tokens["BOT_TOKEN"]}
}
