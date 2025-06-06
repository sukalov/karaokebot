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
	CreatedAt        int64
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
		if err := rows.Scan(&song.ID, &song.Category, &song.Title, &song.Artist, &song.ArtistName, &song.Link, &song.AdditionalChords, &song.Excluded, &song.Counter, &song.CreatedAt); err != nil {
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

func (s *SongbookType) ValidateCategory(category string) bool {
	switch category {
	case "русский рок", "русская поп-музыка", "советское", "детские песни", "зарубежное", "разное":
		return true
	}
	return false
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
            excluded = ?,
            created_at = ?
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
		song.CreatedAt,
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

func (s *SongbookType) NewSong(song Song) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	query := `
        INSERT INTO songbook (
            category, 
            title, 
            link, 
            artist, 
            artist_name, 
            additional_chords, 
            excluded,
            id,
            counter,
            created_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
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
		song.Counter,
		song.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert song: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no row was affected (???): %s", song.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.songs = append(s.songs, song)

	return nil
}

func (s *SongbookType) DeleteSong(songID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	query := `DELETE FROM songbook WHERE id = ?`
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
		return fmt.Errorf("failed to delete song from database: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected after deletion: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no song found with id: %s to delete", songID)
	}

	// Remove from in-memory slice
	s.mu.Lock()
	defer s.mu.Unlock()

	foundIndex := -1
	for i, song := range s.songs {
		if song.ID == songID {
			foundIndex = i
			break
		}
	}

	if foundIndex != -1 {
		s.songs = append(s.songs[:foundIndex], s.songs[foundIndex+1:]...)
	} else {
		log.Printf("Warning: Song with ID %s not found in in-memory cache after successful database deletion.", songID)
	}

	return nil
}

func (s *Song) Stringify(markdown bool) string {
	builder := strings.Builder{}

	builder.WriteString(fmt.Sprintf("ID: %s\n", s.ID))

	if s.Category != "" {
		builder.WriteString(fmt.Sprintf("категория: %s\n", s.Category))
	}

	builder.WriteString(fmt.Sprintf("название: %s\n", s.Title))

	if markdown {
		builder.WriteString(fmt.Sprintf("[ссылка на аккорды](%s)\n", s.Link))
	} else {
		builder.WriteString(fmt.Sprintf("ссылка: %s\n", s.Link))
	}

	if s.Artist.Valid {
		builder.WriteString(fmt.Sprintf("исполнитель: %s\n", s.Artist.String))
	}

	if s.ArtistName.Valid {
		builder.WriteString(fmt.Sprintf("имя исполнителя: %s\n", s.ArtistName.String))
	}

	if s.AdditionalChords.Valid {
		builder.WriteString(fmt.Sprintf("заметка к песне: %s\n", s.AdditionalChords.String))
	}

	if s.Excluded != 0 {
		builder.WriteString("исключена из поиска\n")
	}

	builder.WriteString(fmt.Sprintf("счётчик: %d\n", s.Counter))

	builder.WriteString(fmt.Sprintf("создана: %s", time.Unix(s.CreatedAt, 0).Format("2006-01-02 15:04:05")))

	return builder.String()
}
