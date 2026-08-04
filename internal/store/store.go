package store

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"poker-chip-tracker/internal/models"
)

var (
	ErrRoomNotFound      = errors.New("room not found")
	ErrRoomClosed        = errors.New("room is closed")
	ErrChipUnbalanced    = errors.New("chip changes not balanced")
	ErrPlayerNotFound    = errors.New("player not found")
	ErrInvalidInput      = errors.New("invalid input")
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS rooms (
		id TEXT PRIMARY KEY,
		room_code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'OPEN',
		created_at DATETIME NOT NULL,
		closed_at DATETIME
	);
	CREATE TABLE IF NOT EXISTS players (
		id TEXT PRIMARY KEY,
		room_id TEXT NOT NULL,
		name TEXT NOT NULL,
		initial_chip INTEGER NOT NULL DEFAULT 0,
		current_chip INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY (room_id) REFERENCES rooms(id)
	);
	CREATE INDEX IF NOT EXISTS idx_players_room ON players(room_id);
	CREATE TABLE IF NOT EXISTS game_rounds (
		id TEXT PRIMARY KEY,
		room_id TEXT NOT NULL,
		round_no INTEGER NOT NULL,
		FOREIGN KEY (room_id) REFERENCES rooms(id)
	);
	CREATE INDEX IF NOT EXISTS idx_rounds_room ON game_rounds(room_id);
	CREATE TABLE IF NOT EXISTS chip_records (
		id TEXT PRIMARY KEY,
		round_id TEXT NOT NULL,
		player_id TEXT NOT NULL,
		change_amount INTEGER NOT NULL,
		FOREIGN KEY (round_id) REFERENCES game_rounds(id),
		FOREIGN KEY (player_id) REFERENCES players(id)
	);
	CREATE INDEX IF NOT EXISTS idx_records_round ON chip_records(round_id);
	CREATE TABLE IF NOT EXISTS operation_logs (
		id TEXT PRIMARY KEY,
		room_id TEXT NOT NULL,
		action TEXT NOT NULL,
		detail TEXT,
		created_at DATETIME NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) log(roomID, action, detail string) {
	_, _ = s.db.Exec(
		`INSERT INTO operation_logs (id, room_id, action, detail, created_at) VALUES (?, ?, ?, ?, ?)`,
		uuid.New().String(), roomID, action, detail, time.Now().UTC(),
	)
}

func generateRoomCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		code[i] = chars[n.Int64()]
	}
	return string(code), nil
}

func (s *Store) CreateRoom(name string) (*models.Room, error) {
	name = strings.TrimSpace(name)
	id := uuid.New().String()
	now := time.Now().UTC()

	for i := 0; i < 10; i++ {
		code, err := generateRoomCode()
		if err != nil {
			return nil, err
		}
		_, err = s.db.Exec(
			`INSERT INTO rooms (id, room_code, name, status, created_at) VALUES (?, ?, ?, ?, ?)`,
			id, code, name, models.RoomStatusOpen, now,
		)
		if err == nil {
			s.log(id, "CREATE_ROOM", fmt.Sprintf("code=%s name=%s", code, name))
			return &models.Room{ID: id, RoomCode: code, Name: name, Status: models.RoomStatusOpen, CreatedAt: now}, nil
		}
		if !strings.Contains(err.Error(), "UNIQUE") {
			return nil, err
		}
	}
	return nil, errors.New("failed to generate unique room code")
}

func (s *Store) GetRoomByCode(code string) (*models.Room, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	row := s.db.QueryRow(`SELECT id, room_code, name, status, created_at FROM rooms WHERE room_code = ?`, code)
	var r models.Room
	if err := row.Scan(&r.ID, &r.RoomCode, &r.Name, &r.Status, &r.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (s *Store) CloseRoom(code string) (*models.Room, error) {
	room, err := s.GetRoomByCode(code)
	if err != nil {
		return nil, err
	}
	if room.Status == models.RoomStatusClosed {
		return room, nil
	}
	now := time.Now().UTC()
	_, err = s.db.Exec(`UPDATE rooms SET status = ?, closed_at = ? WHERE id = ?`, models.RoomStatusClosed, now, room.ID)
	if err != nil {
		return nil, err
	}
	s.log(room.ID, "CLOSE_ROOM", "")
	room.Status = models.RoomStatusClosed
	return room, nil
}

func (s *Store) AddPlayer(roomCode, name string, initialChip int) (*models.Player, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidInput
	}
	room, err := s.GetRoomByCode(roomCode)
	if err != nil {
		return nil, err
	}
	if room.Status == models.RoomStatusClosed {
		return nil, ErrRoomClosed
	}
	p := &models.Player{
		ID:          uuid.New().String(),
		RoomID:      room.ID,
		Name:        name,
		InitialChip: initialChip,
		CurrentChip: initialChip,
	}
	_, err = s.db.Exec(
		`INSERT INTO players (id, room_id, name, initial_chip, current_chip) VALUES (?, ?, ?, ?, ?)`,
		p.ID, p.RoomID, p.Name, p.InitialChip, p.CurrentChip,
	)
	if err != nil {
		return nil, err
	}
	s.log(room.ID, "ADD_PLAYER", fmt.Sprintf("name=%s chip=%d", name, initialChip))
	return p, nil
}

func (s *Store) ListPlayers(roomID string) ([]models.Player, error) {
	rows, err := s.db.Query(
		`SELECT id, room_id, name, initial_chip, current_chip FROM players WHERE room_id = ? ORDER BY name`,
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var players = make([]models.Player, 0)
	for rows.Next() {
		var p models.Player
		if err := rows.Scan(&p.ID, &p.RoomID, &p.Name, &p.InitialChip, &p.CurrentChip); err != nil {
			return nil, err
		}
		p.TotalChange = p.CurrentChip - p.InitialChip
		players = append(players, p)
	}
	return players, rows.Err()
}

func (s *Store) CreateRound(roomCode string) (*models.GameRound, error) {
	room, err := s.GetRoomByCode(roomCode)
	if err != nil {
		return nil, err
	}
	if room.Status == models.RoomStatusClosed {
		return nil, ErrRoomClosed
	}
	var maxNo int
	err = s.db.QueryRow(`SELECT COALESCE(MAX(round_no), 0) FROM game_rounds WHERE room_id = ?`, room.ID).Scan(&maxNo)
	if err != nil {
		return nil, err
	}
	round := &models.GameRound{
		ID:      uuid.New().String(),
		RoomID:  room.ID,
		RoundNo: maxNo + 1,
	}
	_, err = s.db.Exec(`INSERT INTO game_rounds (id, room_id, round_no) VALUES (?, ?, ?)`, round.ID, round.RoomID, round.RoundNo)
	if err != nil {
		return nil, err
	}
	s.log(room.ID, "CREATE_ROUND", fmt.Sprintf("round_no=%d", round.RoundNo))
	return round, nil
}

func (s *Store) SubmitRecords(roundID string, inputs []models.RecordInput) (*models.RoundView, error) {
	if len(inputs) == 0 {
		return nil, ErrInvalidInput
	}
	var roomID, roomStatus string
	err := s.db.QueryRow(`
		SELECT r.room_id, rm.status FROM game_rounds r
		JOIN rooms rm ON rm.id = r.room_id
		WHERE r.id = ?`, roundID).Scan(&roomID, &roomStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	if roomStatus == models.RoomStatusClosed {
		return nil, ErrRoomClosed
	}

	sum := 0
	for _, in := range inputs {
		sum += in.ChangeAmount
	}
	if sum != 0 {
		return nil, ErrChipUnbalanced
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var roundNo int
	err = tx.QueryRow(`SELECT round_no FROM game_rounds WHERE id = ?`, roundID).Scan(&roundNo)
	if err != nil {
		return nil, err
	}

	records := make([]models.ChipRecord, 0)
	for _, in := range inputs {
		if in.ChangeAmount == 0 {
			continue
		}
		var playerName string
		err = tx.QueryRow(`SELECT name FROM players WHERE id = ? AND room_id = ?`, in.PlayerID, roomID).Scan(&playerName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrPlayerNotFound
			}
			return nil, err
		}
		recID := uuid.New().String()
		_, err = tx.Exec(
			`INSERT INTO chip_records (id, round_id, player_id, change_amount) VALUES (?, ?, ?, ?)`,
			recID, roundID, in.PlayerID, in.ChangeAmount,
		)
		if err != nil {
			return nil, err
		}
		_, err = tx.Exec(
			`UPDATE players SET current_chip = current_chip + ? WHERE id = ?`,
			in.ChangeAmount, in.PlayerID,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, models.ChipRecord{
			ID: recID, RoundID: roundID, PlayerID: in.PlayerID,
			PlayerName: playerName, ChangeAmount: in.ChangeAmount,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.log(roomID, "SUBMIT_RECORDS", fmt.Sprintf("round=%d count=%d", roundNo, len(records)))

	return &models.RoundView{ID: roundID, RoundNo: roundNo, Records: records}, nil
}

func (s *Store) GetRoomDetail(code string) (*models.RoomDetail, error) {
	room, err := s.GetRoomByCode(code)
	if err != nil {
		return nil, err
	}
	players, err := s.ListPlayers(room.ID)
	if err != nil {
		return nil, err
	}
	rounds, err := s.listRounds(room.ID)
	if err != nil {
		return nil, err
	}
	return &models.RoomDetail{Room: *room, Players: players, Rounds: rounds}, nil
}

func (s *Store) listRounds(roomID string) ([]models.RoundView, error) {
	rows, err := s.db.Query(`SELECT id, round_no FROM game_rounds WHERE room_id = ? ORDER BY round_no DESC`, roomID)
	if err != nil {
		return nil, err
	}
	var rounds []models.RoundView
	for rows.Next() {
		var rv models.RoundView
		if err := rows.Scan(&rv.ID, &rv.RoundNo); err != nil {
			rows.Close()
			return nil, err
		}
		rounds = append(rounds, rv)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	for i := range rounds {
		recs, err := s.listRecords(rounds[i].ID)
		if err != nil {
			return nil, err
		}
		rounds[i].Records = recs
	}
	return rounds, nil
}

func (s *Store) listRecords(roundID string) ([]models.ChipRecord, error) {
	rows, err := s.db.Query(`
		SELECT cr.id, cr.round_id, cr.player_id, p.name, cr.change_amount
		FROM chip_records cr JOIN players p ON p.id = cr.player_id
		WHERE cr.round_id = ?`, roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]models.ChipRecord, 0)
	for rows.Next() {
		var r models.ChipRecord
		if err := rows.Scan(&r.ID, &r.RoundID, &r.PlayerID, &r.PlayerName, &r.ChangeAmount); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) GetStats(code string) (*models.Stats, error) {
	room, err := s.GetRoomByCode(code)
	if err != nil {
		return nil, err
	}
	players, err := s.ListPlayers(room.ID)
	if err != nil {
		return nil, err
	}
	stats := make([]models.PlayerStat, len(players))
	for i, p := range players {
		stats[i] = models.PlayerStat{
			ID: p.ID, Name: p.Name, InitialChip: p.InitialChip,
			CurrentChip: p.CurrentChip, TotalChange: p.CurrentChip - p.InitialChip,
		}
	}
	rankStats(stats)
	return &models.Stats{Players: stats}, nil
}

func rankStats(stats []models.PlayerStat) {
	sort.SliceStable(stats, func(i, j int) bool {
		if stats[i].TotalChange != stats[j].TotalChange {
			return stats[i].TotalChange > stats[j].TotalChange
		}
		return stats[i].Name < stats[j].Name
	})
	for i := range stats {
		stats[i].Rank = i + 1
	}
}

func (s *Store) GetRoomCodeByRoundID(roundID string) (string, error) {
	var code string
	err := s.db.QueryRow(`
		SELECT rm.room_code FROM game_rounds gr
		JOIN rooms rm ON rm.id = gr.room_id
		WHERE gr.id = ?`, roundID).Scan(&code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidInput
		}
		return "", err
	}
	return code, nil
}

func (s *Store) GetHistory(code string) (*models.History, error) {
	room, err := s.GetRoomByCode(code)
	if err != nil {
		return nil, err
	}
	var closedAt sql.NullTime
	_ = s.db.QueryRow(`SELECT closed_at FROM rooms WHERE id = ?`, room.ID).Scan(&closedAt)

	stats, err := s.GetStats(code)
	if err != nil {
		return nil, err
	}
	rounds, err := s.listRounds(room.ID)
	if err != nil {
		return nil, err
	}
	h := &models.History{
		Room: *room, TotalRounds: len(rounds),
		Stats: stats.Players, Rounds: rounds,
	}
	if closedAt.Valid {
		t := closedAt.Time
		h.ClosedAt = &t
	}
	return h, nil
}
