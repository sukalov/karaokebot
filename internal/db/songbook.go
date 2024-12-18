package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
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
	mu    sync.RWMutex
}

var Songbook = &SongbookType{}

func (s *SongbookType) init() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := Database.QueryContext(ctx, "SELECT * FROM songbook")
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		if err := rows.Scan(&song.ID, &song.Category, &song.Title, &song.Artist, &song.ArtistName, &song.Link, &song.AdditionalChords, &song.Excluded, &song.Counter); err != nil {
			log.Printf("error scanning row: %v", err)
			continue
		}
		songs = append(songs, song)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error during rows iteration: %w", err)
	}

	s.songs = songs
	return nil
}

func (s *SongbookType) FindSongByID(id string) (Song, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, song := range s.songs {
		if song.ID == id {
			return song, true
		}
	}
	return Song{}, false
}

func (s *SongbookType) FormatSongName(song Song) string {
	var parts []string
	if song.ArtistName.Valid {
		parts = append(parts, song.ArtistName.String)
	}
	if song.Artist.Valid {
		parts = append(parts, song.Artist.String+" - ")
	}
	parts = append(parts, song.Title)

	return strings.TrimSpace(strings.Join(parts, ""))
}

func (s *SongbookType) IncrementSongCounter(songID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `UPDATE songbook SET counter = counter + 1 WHERE id = ?`
	result, err := Database.ExecContext(ctx, query, songID)
	if err != nil {
		return fmt.Errorf("failed to increment song counter: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no song found with id: %s", songID)
	}

	return nil
}