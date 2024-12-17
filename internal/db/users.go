package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type User struct {
	ID             int64
	ChatID         int64
	Username       sql.NullString
	TgName         sql.NullString
	SavedName      sql.NullString
	AddedAt        time.Time
	TimesPerformed int
}

func RegisterUser(update tgbotapi.Update) error {
	// Prepare context and ensure database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Prepare user data
	message := update.Message
	userName := sql.NullString{
		String: message.From.UserName,
		Valid:  message.From.UserName != "",
	}
	tgName := sql.NullString{
		String: message.From.FirstName + " " + message.From.LastName,
		Valid:  message.From.FirstName+message.From.LastName != "",
	}
	typedName := sql.NullString{
		Valid: false,
	}

	// Check if user already exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE chat_id = ?)`
	err := Database.QueryRowContext(ctx, checkQuery, message.Chat.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking user existence: %v", err)
	}

	// If user doesn't exist, insert new user
	if !exists {
		insertQuery := `
			INSERT INTO users (
				chat_id,
				username,
				tg_name,
				saved_name,
				added_at,
				times_performed
			) VALUES (?, ?, ?, ?, ?, ?)
		`

		_, err = Database.ExecContext(ctx, insertQuery,
			message.Chat.ID,
			userName,
			tgName,
			typedName,
			time.Now(),
			0,
		)
		if err != nil {
			return fmt.Errorf("failed to insert new user: %v", err)
		}

		log.Printf("new user registered: ID: %d, username: %s",
			message.Chat.ID,
			userName.String,
		)
	}

	return nil
}
