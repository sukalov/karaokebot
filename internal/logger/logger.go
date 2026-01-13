package logger

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/sukalov/karaokebot/internal/utils"
	"github.com/sukalov/karaokebot/internal/utils/e"
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

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s\n%s", timestamp, prefix, message)

	go func() {
		if err := botClient.SendMessage(ChannelID, logMessage); err != nil {
			fmt.Printf("Failed to send log to channel: %v\nLog was: %s\n", err, logMessage)
		}
	}()
}

func LogWithErr(message string, err error) {
	if err == nil {
		Info(message)
		return
	}

	msg := fmt.Sprintf("%s\nError: %v", message, err)
	Error(msg)

	e.Wrap(message, err)
}
