package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
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

type SongbookType struct {
	songs []Song
}

var Songbook = &SongbookType{}

func init() {
	Songbook.getSongbook(Database)
}

func (s *SongbookType) FindSongByID(id string) (Song, bool) {
	for _, song := range s.songs {
		if song.ID == id {
			return song, true
		}
	}
	return Song{}, false
}

func (s *SongbookType) FormatSongName(song Song) string {
	artistName := ""
	if song.ArtistName.Valid {
		artistName = song.ArtistName.String
	}
	artist := ""
	if song.Artist.Valid {
		artist = song.Artist.String + " - "
	}

	return strings.TrimSpace(fmt.Sprintf("%s %s%s", artistName, artist, song.Title))
}

func (s *SongbookType) getSongbook(db *sql.DB) {
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

		s.songs = append(s.songs, song)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("error during rows iteration:", err)
	}
}

func (s *SongbookType) IncrementSongCounter(songID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `UPDATE songbook SET counter = counter + 1 WHERE id = ?`

	_, err := Database.ExecContext(ctx, query, songID)
	if err != nil {
		return fmt.Errorf("failed to increment song counter: %v", err)
	}

	return nil
}
