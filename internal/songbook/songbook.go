package songbook

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/sukalov/karaokebot/internal/utils"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Song struct {
	ID               string
	Category         string
	Title            string
	Link             string
	Artist           sql.NullString
	ArtistName       sql.NullString
	AdditionalChords sql.NullString
}

var songs []Song

func init() {
	env, err := utils.LoadEnv([]string{"TURSO_DATABASE_URL", "TURSO_AUTH_TOKEN"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load db env %s.", err)
		os.Exit(1)
	}
	url := fmt.Sprintf("%s?authToken=%s", env["TURSO_DATABASE_URL"], env["TURSO_AUTH_TOKEN"])

	db, err := sql.Open("libsql", url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s: %s", url, err)
		os.Exit(1)
	}
	getSongbook(db)
	defer db.Close()
}

func FindSongByID(id string) (Song, bool) {
	for _, song := range songs {
		if song.ID == id {
			return song, true
		}
	}
	return Song{}, false
}

func FormatSongName(s Song) string {
	artistName := ""
	if s.ArtistName.Valid {
		artistName = s.ArtistName.String
	}
	artist := "неизвествен"
	if s.Artist.Valid {
		artist = s.Artist.String
	}

	return strings.TrimSpace(fmt.Sprintf("%s %s - %s", artistName, artist, s.Title))
}

func getSongbook(db *sql.DB) {
	rows, err := db.Query("SELECT * FROM songbook")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to execute query: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		var song Song

		if err := rows.Scan(&song.ID, &song.Category, &song.Title, &song.Artist, &song.ArtistName, &song.Link, &song.AdditionalChords); err != nil {
			fmt.Println("error scanning row:", err)
			return
		}

		songs = append(songs, song)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("error during rows iteration:", err)
	}
}
