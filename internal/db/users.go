package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sukalov/karaokebot/internal/logger"
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
			logger.Error(fmt.Sprintf("🎵🔴 [ERROR] Query timeout after 5 seconds\nChat ID: %d", update.Message.Chat.ID))
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
				logger.Error(fmt.Sprintf("🎵🔴 [ERROR] Error rolling back transaction\nChat ID: %d\nError: %v", message.Chat.ID, rollbackErr))
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				logger.Error(fmt.Sprintf("🎵🔴 [ERROR] Error committing transaction\nChat ID: %d\nError: %v", message.Chat.ID, commitErr))
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

		logger.Info(fmt.Sprintf("🎵📋 [INFO] New user registered\nChat ID: %d\nUsername: %s\nName: %s",
			message.Chat.ID,
			userName.String,
			tgName.String,
		))
	}

	return nil
}

func (u *UsersType) GetByChatID(chatID int64) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			logger.Error(fmt.Sprintf("🎵🔴 [ERROR] Query timeout after 5 seconds\nChat ID: %d", chatID))
		}
	}()

	var user User

	var timestampStr string
	query := `SELECT chat_id, username, tg_name, saved_name, added_at, times_performed
			  FROM users WHERE chat_id = ?`
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
	defer func() {
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			logger.Error(fmt.Sprintf("🎵🔴 [ERROR] Query timeout after 5 seconds\nChat ID: %d", chatID))
		}
	}()

	query := `UPDATE users SET saved_name = ? WHERE chat_id = ?`
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
