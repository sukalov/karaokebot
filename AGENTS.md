# Karaokebot - Agent Development Guide

This guide helps agentic coding agents work effectively in the karaokebot repository.

## Build, Lint, and Test Commands

### Build Commands
- `make build` - Build the binary (CGO_ENABLED=0 GOOS=linux)
- `make dev` - Run with Air hot reload during development
- `make deps` - Download Go dependencies
- `go run ./cmd/karaokebot` - Run directly without building

### Linting
- `trunk check` - Run all linters (includes gofmt, golangci-lint, etc.)
- `gofmt -s -w .` - Format Go code
- `golangci-lint run` - Run Go linter specifically

### Testing
- `go test ./...` - Run all tests
- `go test ./internal/package` - Run tests for a specific package
- `go test -run TestFunctionName ./internal/package` - Run a single test
- `go test -v ./internal/package` - Run tests with verbose output

## Code Style Guidelines

### Imports
- Group imports: standard library, external packages, internal packages
- Separate groups with blank lines
- Use short aliases for external packages (e.g., `tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"`)
- Example:
  ```go
  import (
      "context"
      "fmt"

      tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
      "github.com/sukalov/karaokebot/internal/bot"
  )
  ```

### Formatting
- Use `gofmt -s` for formatting (enforced by trunk)
- No explicit line length limit, but keep code readable
- Use tabs for indentation (Go standard)

### Types
- Exported struct fields use PascalCase
- JSON struct tags use snake_case: `json:"song_id"`
- Interface types should be simple, descriptive names (e.g., `BotClient`)
- Type definitions use PascalCase: `type UserState struct`

### Naming Conventions
- Packages: lowercase, single words (e.g., `state`, `utils`)
- Functions: PascalCase for exported, camelCase for unexported
- Variables: camelCase for local, PascalCase for exported
- Constants: UPPER_SNAKE_CASE (e.g., `StageAskingName`)
- Handler names end with "Handler" (e.g., `lineHandler`)

### Error Handling
- Always return errors, don't panic unless unrecoverable
- Use `fmt.Errorf("%s: %w", msg, err)` for error wrapping
- Use `e.Wrap(msg, err)` from `internal/utils/e` for consistent wrapping
- Log errors with `logger.Error(fmt.Sprintf("context: %v", err))`
- In handlers, return error to be handled by caller

### Logging
- Use `logger.Info(isAdmin, message)` for informational messages
- Use `logger.Error(isAdmin, message)` for errors
- Use `logger.Debug(isAdmin, message)` for debugging information
- Use `logger.Success(isAdmin, message)` for successful operations
- First parameter: `true` for admin logs, `false` for client logs
- All emojis are handled by logger - DO NOT include emojis in log messages
- Log format: `🎵 ℹ️ INFO Message` or `⚙️ ❌ ERROR Message`

### Handlers
- Handler signature: `func(b *bot.Bot, update tgbotapi.Update) error`
- Use `update.Message.Command()` for command handling
- Use `update.CallbackQuery.Data` for inline button callbacks
- Return `bot.ErrMessageHandled` to stop message processing
- Return nil after successful handling

### Database and Redis
- Package-level init functions for initialization
- Use singleton pattern with `sync.Once`
- Context parameter required for Redis operations: `ctx context.Context`
- Pass context.Background() when no specific context needed
- Connection settings: 25 max open/idle connections, 5 min lifetime

### Concurrency
- Use `sync.RWMutex` for state management in StateManager
- Lock before modifying shared state: `defer mu.Unlock()`
- Use read locks for reads: `mu.RLock()` / `defer mu.RUnlock()`
- Use goroutines for non-blocking operations (e.g., logging)

### Environment Variables
- Use `utils.LoadEnv([]string{"VAR_NAME"})` to load required env vars
- Required vars: BOT_TOKEN, ADMIN_BOT_TOKEN, REDIS_URL, REDIS_PASSWORD, TURSO_DATABASE_URL, TURSO_AUTH_TOKEN, LOG_CHANNEL_ID
- Never commit .env file (in .gitignore)

### Project Structure
- `cmd/` - Application entry points
- `internal/bot/` - Bot core (admin, client, common)
- `internal/state/` - State management with Redis sync
- `internal/db/` - Database (Turso/SQLite) operations
- `internal/redis/` - Redis client and functions
- `internal/lyrics/` - Lyrics parsing service
- `internal/logger/` - Logging to Telegram channel
- `internal/users/` - User state types
- `internal/utils/` - Utilities (env loading, errors)

### Best Practices
- Use context for Redis operations
- Validate user input in handlers before processing
- Use `strings.TrimSpace()` to clean user input
- Check admin permissions: `if !h.admins[username] { return "not admin" }`
- Store timestamps in time.Time, format for display
- Use `time.FixedZone()` for timezone handling (e.g., Moscow Time)
