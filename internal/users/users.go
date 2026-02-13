package users

import (
	"time"
)

type UserState struct {
	ID         int       `json:"id"`
	ChatID     int64     `json:"chat_id"`
	Username   string    `json:"username"`
	TgName     string    `json:"tg_name"`
	SongID     string    `json:"song_id"`
	SongName   string    `json:"song_name"`
	SongLink   string    `json:"song_link"`
	SongNote   string    `json:"additional_chords"`
	Stage      string    `json:"stage"`
	TimeAdded  time.Time `json:"time_added"`
	TypedName  string    `json:"typed_name"`
	LyricsURL  string    `json:"lyrics_url"`
	LyricsText string    `json:"lyrics_text"`
}

const (
	StageAskingName      = "asking_name"
	StageAwaitingPayment = "awaiting_payment"
	StageInLine          = "in_line"
)
