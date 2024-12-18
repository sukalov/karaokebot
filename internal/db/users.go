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
	ChatID         int64
	Username       sql.NullString
	TgName         sql.NullString
	SavedName      sql.NullString
	AddedAt        time.Time
	TimesPerformed int
}

type UsersType struct{}

var Users = &UsersType{}

func init() {
	// If you need to load users initially, you can add a method to do so here
}

func (u *UsersType) Register(update tgbotapi.Update) error {
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

func (u *UsersType) GetByChatID(chatID int64) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	query := `SELECT chat_id, username, tg_name, saved_name, added_at, times_performed
			  FROM users WHERE chat_id = ?`

	var timestampStr string
	err := Database.QueryRowContext(ctx, query, chatID).Scan(
		&user.ChatID,
		&user.Username,
		&user.TgName,
		&user.SavedName,
		&timestampStr,
		&user.TimesPerformed,
	)
	if err != nil {
		// Handle error
		return User{}, err
	}

	// Parse the timestamp string manually
	user.AddedAt, err = time.Parse("2006-01-02 15:04:05.999999-07:00", timestampStr)
	if err != nil {
		// Handle parsing error
		return User{}, err
	}

	return user, nil
}

func (u *UsersType) UpdateSavedName(chatID int64, newName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE users SET saved_name = ? WHERE chat_id = ?`

	_, err := Database.ExecContext(ctx, query, newName, chatID)
	if err != nil {
		return fmt.Errorf("failed to update saved name: %v", err)
	}

	return nil
}

// New method to increment times performed for a user
func (u *UsersType) IncrementTimesPerformed(chatID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE users SET times_performed = times_performed + 1 WHERE chat_id = ?`

	_, err := Database.ExecContext(ctx, query, chatID)
	if err != nil {
		return fmt.Errorf("failed to increment times performed: %v", err)
	}

	return nil
}
