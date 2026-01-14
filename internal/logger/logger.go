package logger

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/sukalov/karaokebot/internal/utils"
)

var (
	ChannelID int64
	once      sync.Once
	botClient BotClient
)

type BotClient interface {
	SendMessage(chatID int64, text string) error
}

func Init(client BotClient) error {
	var initErr error
	once.Do(func() {
		env, err := utils.LoadEnv([]string{"LOG_CHANNEL_ID"})
		if err != nil {
			initErr = fmt.Errorf("failed to load LOG_CHANNEL_ID: %w", err)
			return
		}

		ChannelID, err = strconv.ParseInt(env["LOG_CHANNEL_ID"], 10, 64)
		if err != nil {
			initErr = fmt.Errorf("failed to parse LOG_CHANNEL_ID: %w", err)
			return
		}

		botClient = client
	})

	return initErr
}

func Info(isAdmin bool, message string) {
	sendLog(isAdmin, "ℹ️ INFO", message)
}

func Error(isAdmin bool, message string) {
	sendLog(isAdmin, "❌ ERROR", message)
}

func Debug(isAdmin bool, message string) {
	sendLog(isAdmin, "🔍 DEBUG", message)
}

func Success(isAdmin bool, message string) {
	sendLog(isAdmin, "✅ SUCCESS", message)
}

func sendLog(isAdmin bool, prefix, message string) {
	if botClient == nil {
		return
	}

	botEmoji := "🎵"
	if isAdmin {
		botEmoji = "⚙️"
	}

	logMessage := fmt.Sprintf("%s %s %s", botEmoji, prefix, message)

	go func() {
		if err := botClient.SendMessage(ChannelID, logMessage); err != nil {
			fmt.Printf("Failed to send log to channel: %v\nLog was: %s\n", err, logMessage)
		}
	}()
}
