package main

import (
	"log"
	"sync"

	"github.com/sukalov/karaokebot/internal/bot"
	"github.com/sukalov/karaokebot/internal/bot/admin"
	"github.com/sukalov/karaokebot/internal/bot/client"
	"github.com/sukalov/karaokebot/internal/state"
	"github.com/sukalov/karaokebot/internal/utils"
)

// 	// admins        = []string{"sukalov", "olakotr", "yatskovanatoly", "motygapishet"}

type Tokens struct {
	clientBotToken string
	adminBotToken  string
}

func main() {
	// Create wait group to keep main thread alive
	var wg sync.WaitGroup
	wg.Add(1)

	userManager := state.NewStateManager()
	userManager.Init()
	adminUsernames := []string{"sukalov", "olakotr", "motygapishet"}
	tokens := mustTokens()

	adminBot, err := bot.New("AdminBot", tokens.adminBotToken)
	if err != nil {
		log.Fatalf("failed to create admin bot: %v", err)
	}

	clientBot, err := bot.New("ClientBot", tokens.clientBotToken)
	if err != nil {
		log.Fatalf("failed to create client bot: %v", err)
	}

	// Setup and start admin bot handlers
	admin.SetupHandlers(adminBot, userManager, adminUsernames)

	// Setup and start client bot handlers
	client.SetupHandlers(clientBot, userManager)

	// Wait indefinitely
	wg.Wait()
}

func mustTokens() Tokens {
	tokens, err := utils.LoadEnv([]string{"BOT_TOKEN", "ADMIN_BOT_TOKEN"})

	if err != nil {
		log.Fatal("required env missing")
	}

	return Tokens{adminBotToken: tokens["ADMIN_BOT_TOKEN"], clientBotToken: tokens["BOT_TOKEN"]}
}
