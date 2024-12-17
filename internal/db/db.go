package db

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/sukalov/karaokebot/internal/utils"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

var Database *sql.DB

func init() {
	env, err := utils.LoadEnv([]string{"TURSO_DATABASE_URL", "TURSO_AUTH_TOKEN"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load db env %s.", err)
		os.Exit(1)
	}
	url := fmt.Sprintf("%s?authToken=%s", env["TURSO_DATABASE_URL"], env["TURSO_AUTH_TOKEN"])

	Database, err = sql.Open("libsql", url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s: %s", url, err)
		os.Exit(1)
	}
}

// Close closes the database connection
func Close() {
	if Database != nil {
		Database.Close()
	}
}
