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
		parts = append(parts, fmt.Sprint(song.ArtistName.String, " "))
	}
	if song.Artist.Valid {
		parts = append(parts, song.Artist.String+" - ")
	}
	parts = append(parts, song.Title)

	return strings.TrimSpace(strings.Join(parts, ""))
}

func (s *SongbookType) IncrementSongCounter(songID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	query := `UPDATE songbook SET counter = counter + 1 WHERE id = ?`
	defer func() {
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("error: timeout in query '%s' canceled after 5 seconds: %v",
				query,
				ctx.Err(),
			)
		}
	}()

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

func (s *SongbookType) SearchSongs(query string) []Song {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	queryParts := strings.Fields(query)

	var results []Song

	for _, song := range s.songs {
		title := strings.ToLower(song.Title)
		artist := strings.ToLower(song.Artist.String)
		artistName := strings.ToLower(song.ArtistName.String)

		// Check if all query parts match either title or artist
		allPartsMatch := true
		for _, part := range queryParts {
			matches := strings.Contains(title, part) ||
				strings.Contains(artist, part) ||
				strings.Contains(artistName, part)

			if !matches {
				allPartsMatch = false
				break
			}
		}

		if allPartsMatch {
			results = append(results, song)
		}
	}

	return results
}

func (s *SongbookType) UpdateSong(song Song) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	fmt.Print("START UPDATE")
	query := `
		UPDATE songbook 
		SET category = ?, 
			title = ?, 
			link = ?, 
			artist = ?, 
			artist_name = ?, 
			additional_chords = ?, 
			excluded = ?
		WHERE id = ?`
	defer func() {
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("error: timeout in query '%s' canceled after 5 seconds: %v",
				query,
				ctx.Err(),
			)
		}
	}()

	result, err := Database.ExecContext(ctx, query,
		song.Category,
		song.Title,
		song.Link,
		song.Artist,
		song.ArtistName,
		song.AdditionalChords,
		song.Excluded,
		song.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update song: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no song found with id: %s", song.ID)
	}

	// Update the in-memory array
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existingSong := range s.songs {
		if existingSong.ID == song.ID {
			song.Counter = existingSong.Counter
			s.songs[i] = song
			return nil
		}
	}

	return fmt.Errorf("song not found in memory: %s", song.ID)
}
