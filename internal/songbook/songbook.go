package songbook

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gocarina/gocsv"
)

type Song struct {
	Rubric      string `csv:"rubric"`
	Artist      string `csv:"artist"`
	Song        string `csv:"song"`
	Name        string `csv:"name"`
	Link        string `csv:"link"`
	ExtraChords string `csv:"extra_chords"`
	ID          string `csv:"id"`
}

var songs []Song
var songsErr error

func init() {
	songs, songsErr = readSongsCSV("data/songbook.csv")
	if songsErr != nil {
		log.Fatalf("failed to load songs: %v", songsErr)
	}
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
	return strings.TrimSpace(fmt.Sprintf("%s %s - %s", s.Name, s.Artist, s.Song))
}

func readSongsCSV(filePath string) ([]Song, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var songs []Song
	if err := gocsv.UnmarshalFile(file, &songs); err != nil {
		return nil, err
	}

	return songs, nil
}
