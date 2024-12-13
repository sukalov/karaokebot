package bot

import (
	"log"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
	log.Printf("[%s] authorized on account %s", b.name, b.Client.Self.UserName)

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
				log.Printf("[%s] command handler error: %v", b.name, err)
			}
			return
		}
	}

	// Handle callback queries
	if update.CallbackQuery != nil {
		if handler, exists := callbackHandlers[update.CallbackQuery.Data]; exists {
			if err := handler(b, update); err != nil {
				log.Printf("[%s] callback handler error: %v", b.name, err)
			}
			return
		}
	}

	// Run generic message handlers
	for _, handler := range messageHandlers {
		if err := handler(b, update); err != nil {
			log.Printf("[%s] message handler error: %v", b.name, err)
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
