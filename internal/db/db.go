// db.go
package db

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sukalov/karaokebot/internal/utils"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

var (
	Database *sql.DB
	once     sync.Once
	initErr  error
)

// MustInit initializes the database and panics if initialization fails
func init() {
	once.Do(func() {
		env, err := utils.LoadEnv([]string{"TURSO_DATABASE_URL", "TURSO_AUTH_TOKEN"})
		if err != nil {
			initErr = fmt.Errorf("failed to load db env: %w", err)
			log.Fatalf("database initialization failed: %v", initErr)
		}
		url := fmt.Sprintf("%s?authToken=%s", env["TURSO_DATABASE_URL"], env["TURSO_AUTH_TOKEN"])

		Database, initErr = sql.Open("libsql", url)
		if initErr != nil {
			log.Fatalf("failed to open db %s: %v", url, initErr)
		}

		// Add connection pool configuration
		Database.SetMaxOpenConns(25)
		Database.SetMaxIdleConns(25)
		Database.SetConnMaxLifetime(5 * time.Minute)

		// Verify database connection
		if pingErr := Database.Ping(); pingErr != nil {
			log.Fatalf("failed to ping database: %v", pingErr)
		}
	})

	// Songbook and other initializations
	if err := Songbook.init(); err != nil {
		log.Fatalf("failed to initialize songbook: %v", err)
	}
}

// Close closes the database connection safely
func Close() {
	if Database != nil {
		if err := Database.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}
}
