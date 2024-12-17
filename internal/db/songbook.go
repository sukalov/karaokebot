package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

type Song struct {
	ID               string
	Category         string
	Title            string
	Link             string
	Artist           sql.NullString
	ArtistName       sql.NullString
	AdditionalChords sql.NullString
	Excluded         int
	Counter          int
}

var songs []Song

func init() {
	getSongbook(Database)
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
	artist := ""
	if s.Artist.Valid {
		artist = s.Artist.String + " - "
	}

	return strings.TrimSpace(fmt.Sprintf("%s %s%s", artistName, artist, s.Title))
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

		if err := rows.Scan(&song.ID, &song.Category, &song.Title, &song.Artist, &song.ArtistName, &song.Link, &song.AdditionalChords, &song.Excluded, &song.Counter); err != nil {
			fmt.Println("error scanning row: ", err)
			return
		}

		songs = append(songs, song)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("error during rows iteration:", err)
	}
}
