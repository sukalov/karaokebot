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

func Info(message string) {
	sendLog("ℹ️ INFO", message)
}

func Error(message string) {
	sendLog("❌ ERROR", message)
}

func Debug(message string) {
	sendLog("🔍 DEBUG", message)
}

func Success(message string) {
	sendLog("✅ SUCCESS", message)
}

func sendLog(prefix, message string) {
	if botClient == nil {
		return
	}

	logMessage := fmt.Sprintf("%s %s", prefix, message)

	go func() {
		if err := botClient.SendMessage(ChannelID, logMessage); err != nil {
			fmt.Printf("Failed to send log to channel: %v\nLog was: %s\n", err, logMessage)
		}
	}()
}
