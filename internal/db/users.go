package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
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

func (u *UsersType) Register(update tgbotapi.Update) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("error: timeout in 'Register' canceled after 5 seconds: %v",
				ctx.Err(),
			)
		}
	}()

	message := update.Message
	userName := sql.NullString{
		String: message.From.UserName,
		Valid:  message.From.UserName != "",
	}
	tgName := sql.NullString{
		String: strings.TrimSpace(message.From.FirstName + " " + message.From.LastName),
		Valid:  message.From.FirstName+message.From.LastName != "",
	}
	typedName := sql.NullString{
		Valid: false,
	}

	tx, err := Database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("error rolling back transaction: %v", rollbackErr)
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("error committing transaction: %v", commitErr)
			}
		}
	}()

	// Use transaction for atomic operations
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM users WHERE chat_id = ?)`
	if err = tx.QueryRowContext(ctx, checkQuery, message.Chat.ID).Scan(&exists); err != nil {
		return fmt.Errorf("error checking user existence: %w", err)
	}

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

		_, err = tx.ExecContext(ctx, insertQuery,
			message.Chat.ID,
			userName,
			tgName,
			typedName,
			time.Now(),
			0,
		)
		if err != nil {
			return fmt.Errorf("failed to insert new user: %w", err)
		}

		log.Printf("new user registered: id: %d, username: %s",
			message.Chat.ID,
			userName.String,
		)
	}

	return nil
}

func (u *UsersType) GetByChatID(chatID int64) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	query := `SELECT chat_id, username, tg_name, saved_name, added_at, times_performed
			  FROM users WHERE chat_id = ?`
	defer func() {
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("error: timeout in query '%s' canceled after 5 seconds: %v",
				query,
				ctx.Err(),
			)
		}
	}()

	var user User

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
		if err == sql.ErrNoRows {
			return User{}, fmt.Errorf("user not found: %d", chatID)
		}
		return User{}, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return user, nil
}

func (u *UsersType) UpdateSavedName(chatID int64, newName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	query := `UPDATE users SET saved_name = ? WHERE chat_id = ?`
	defer func() {
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("error: timeout in query '%s' canceled after 5 seconds: %v",
				query,
				ctx.Err(),
			)
		}
	}()

	result, err := Database.ExecContext(ctx, query, newName, chatID)
	if err != nil {
		return fmt.Errorf("failed to update saved name: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no user found with chat id: %d", chatID)
	}

	return nil
}

func (u *UsersType) IncrementTimesPerformed(chatID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE users SET times_performed = times_performed + 1 WHERE chat_id = ?`

	result, err := Database.ExecContext(ctx, query, chatID)
	if err != nil {
		return fmt.Errorf("failed to increment times performed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no user found with chat id: %d", chatID)
	}

	return nil
}
