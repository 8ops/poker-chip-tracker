package models

import "time"

const (
	RoomStatusOpen   = "OPEN"
	RoomStatusClosed = "CLOSED"
)

type Room struct {
	ID        string    `json:"id"`
	RoomCode  string    `json:"roomCode"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type Player struct {
	ID           string `json:"id"`
	RoomID       string `json:"roomId"`
	Name         string `json:"name"`
	InitialChip  int    `json:"initialChip"`
	CurrentChip  int    `json:"currentChip"`
	TotalChange  int    `json:"totalChange,omitempty"`
}

type GameRound struct {
	ID      string `json:"id"`
	RoomID  string `json:"roomId"`
	RoundNo int    `json:"roundNo"`
}

type ChipRecord struct {
	ID           string `json:"id"`
	RoundID      string `json:"roundId"`
	PlayerID     string `json:"playerId"`
	PlayerName   string `json:"playerName,omitempty"`
	ChangeAmount int    `json:"changeAmount"`
}

type RoomDetail struct {
	Room    Room        `json:"room"`
	Players []Player    `json:"players"`
	Rounds  []RoundView `json:"rounds"`
}

type RoundView struct {
	ID      string       `json:"id"`
	RoundNo int          `json:"roundNo"`
	Records []ChipRecord `json:"records"`
}

type Stats struct {
	Players []PlayerStat `json:"players"`
}

type PlayerStat struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	InitialChip int    `json:"initialChip"`
	CurrentChip int    `json:"currentChip"`
	TotalChange int    `json:"totalChange"`
	Rank        int    `json:"rank"`
}

type History struct {
	Room       Room         `json:"room"`
	TotalRounds int         `json:"totalRounds"`
	ClosedAt   *time.Time   `json:"closedAt,omitempty"`
	Stats      []PlayerStat `json:"stats"`
	Rounds     []RoundView  `json:"rounds"`
}

type CreateRoomRequest struct {
	Name string `json:"name"`
}

type CreateRoomResponse struct {
	RoomCode string `json:"roomCode"`
	RoomID   string `json:"roomId"`
}

type AddPlayerRequest struct {
	RoomCode     string `json:"roomCode"`
	Name         string `json:"name"`
	InitialChip  int    `json:"initialChip"`
}

type SubmitRecordsRequest struct {
	Records []RecordInput `json:"records"`
}

type RecordInput struct {
	PlayerID     string `json:"playerId"`
	ChangeAmount int    `json:"changeAmount"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
