// Package bot provides core functionality for the karaoke telegram bot.
package bot

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/logger"
)

// ErrMessageHandled indicates a message was handled and processing should stop
var ErrMessageHandled = errors.New("message handled, stop processing")

// Bot represents a configurable Telegram bot
type Bot struct {
	Client     *tgbotapi.BotAPI
	updateChan tgbotapi.UpdatesChannel
	stopChan   chan struct{}
	name       string
	mu         sync.Mutex
}

// New creates a new bot instance
func New(name, token string) (*Bot, error) {
	// Create bot client
	botClient, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	// Configure update configuration
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updateChan := botClient.GetUpdatesChan(updateConfig)

	return &Bot{
		Client:     botClient,
		updateChan: updateChan,
		stopChan:   make(chan struct{}),
		name:       name,
	}, nil
}

// Start begins processing updates with custom handler
func (b *Bot) Start(
	commandHandlers map[string]func(b *Bot, update tgbotapi.Update) error,
	messageHandlers []func(b *Bot, update tgbotapi.Update) error,
	callbackHandlers map[string]func(b *Bot, update tgbotapi.Update) error,
) {
	logger.Info(strings.Contains(b.name, "admin"), fmt.Sprintf("%s authorized on account %s", b.name, b.Client.Self.UserName))

	for {
		select {
		case update := <-b.updateChan:
			go b.processUpdate(update, commandHandlers, messageHandlers, callbackHandlers)
		case <-b.stopChan:
			return
		}
	}
}

// processUpdate handles incoming updates with custom handlers
func (b *Bot) processUpdate(
	update tgbotapi.Update,
	commandHandlers map[string]func(b *Bot, update tgbotapi.Update) error,
	messageHandlers []func(b *Bot, update tgbotapi.Update) error,
	callbackHandlers map[string]func(b *Bot, update tgbotapi.Update) error,
) {
	// Handle command updates
	if update.Message != nil && update.Message.IsCommand() {
		if handler, exists := commandHandlers[update.Message.Command()]; exists {
			if err := handler(b, update); err != nil {
				logger.Error(strings.Contains(b.name, "admin"), fmt.Sprintf("[%s] command handler error: %v", b.name, err))
			}
			return
		}
	}

	// Handle callback queries
	if update.CallbackQuery != nil {
		parts := strings.SplitN(update.CallbackQuery.Data, ":", 3)
		var query string
		if len(parts) > 1 {
			query = parts[0]
		} else {
			query = update.CallbackQuery.Data
		}
		if handler, exists := callbackHandlers[query]; exists {
			if err := handler(b, update); err != nil {
				logger.Error(strings.Contains(b.name, "admin"), fmt.Sprintf("[%s] callback handler error: %v", b.name, err))
			}
			return
		} else {
			b.SendMessage(update.Message.Chat.ID, "команда не распознана")
		}
	}

	// Run generic message handlers
	for _, handler := range messageHandlers {
		if err := handler(b, update); err != nil {
			if errors.Is(err, ErrMessageHandled) {
				break
			}
			logger.Error(strings.Contains(b.name, "admin"), fmt.Sprintf("[%s] message handler error: %v", b.name, err))
		}
	}
}

// Stop halts the bot
func (b *Bot) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.stopChan <- struct{}{}
}

func (b *Bot) SendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.DisableWebPagePreview = true
	_, err := b.Client.Send(msg)
	return err
}

func (b *Bot) SendMessageWithMarkdown(chatID int64, text string, disableLinks bool) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = disableLinks
	_, err := b.Client.Send(msg)
	return err
}

func (b *Bot) SendMessageWithButtons(
	chatID int64,
	text string,
	keyboard tgbotapi.InlineKeyboardMarkup,
) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	_, err := b.Client.Send(msg)
	return err
}

func (b *Bot) SendMessageWithButtonsNoLinks(
	chatID int64,
	text string,
	keyboard tgbotapi.InlineKeyboardMarkup,
) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	_, err := b.Client.Send(msg)
	return err
}
