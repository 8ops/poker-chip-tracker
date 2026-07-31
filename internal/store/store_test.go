package store

import (
	"path/filepath"
	"testing"

	"poker-chip-tracker/internal/models"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestCreateRoom(t *testing.T) {
	s := tempStore(t)
	room, err := s.CreateRoom("周末棋牌")
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if room.RoomCode == "" || len(room.RoomCode) != 6 {
		t.Fatalf("unexpected room code: %q", room.RoomCode)
	}
	if room.Status != models.RoomStatusOpen {
		t.Fatalf("status = %s, want OPEN", room.Status)
	}
	if room.Name != "周末棋牌" {
		t.Fatalf("name = %s", room.Name)
	}

	got, err := s.GetRoomByCode(room.RoomCode)
	if err != nil {
		t.Fatalf("GetRoomByCode: %v", err)
	}
	if got.ID != room.ID {
		t.Fatalf("id mismatch")
	}
}

func TestCreateRoomUniqueCodes(t *testing.T) {
	s := tempStore(t)
	codes := map[string]struct{}{}
	for i := 0; i < 20; i++ {
		room, err := s.CreateRoom("")
		if err != nil {
			t.Fatalf("CreateRoom: %v", err)
		}
		if _, ok := codes[room.RoomCode]; ok {
			t.Fatalf("duplicate room code: %s", room.RoomCode)
		}
		codes[room.RoomCode] = struct{}{}
	}
}

func TestJoinRoomNotFound(t *testing.T) {
	s := tempStore(t)
	_, err := s.GetRoomByCode("NOTEXS")
	if err != ErrRoomNotFound {
		t.Fatalf("err = %v, want ErrRoomNotFound", err)
	}
}

func TestAddPlayerAndList(t *testing.T) {
	s := tempStore(t)
	room, err := s.CreateRoom("test")
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.AddPlayer(room.RoomCode, "Alice", 1000)
	if err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	if p.CurrentChip != 1000 || p.InitialChip != 1000 {
		t.Fatalf("chip mismatch: %+v", p)
	}

	players, err := s.ListPlayers(room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != 1 || players[0].Name != "Alice" {
		t.Fatalf("players = %+v", players)
	}
}

func TestAddPlayerClosedRoom(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("test")
	_, _ = s.CloseRoom(room.RoomCode)
	_, err := s.AddPlayer(room.RoomCode, "Bob", 500)
	if err != ErrRoomClosed {
		t.Fatalf("err = %v, want ErrRoomClosed", err)
	}
}

func TestAddPlayerEmptyName(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("test")
	_, err := s.AddPlayer(room.RoomCode, "  ", 100)
	if err != ErrInvalidInput {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}

func TestSubmitRecordsBalance(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("test")
	a, _ := s.AddPlayer(room.RoomCode, "Alice", 1000)
	b, _ := s.AddPlayer(room.RoomCode, "Bob", 1000)
	round, err := s.CreateRound(room.RoomCode)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.SubmitRecords(round.ID, []models.RecordInput{
		{PlayerID: a.ID, ChangeAmount: 100},
		{PlayerID: b.ID, ChangeAmount: -50},
	})
	if err != ErrChipUnbalanced {
		t.Fatalf("err = %v, want ErrChipUnbalanced", err)
	}

	view, err := s.SubmitRecords(round.ID, []models.RecordInput{
		{PlayerID: a.ID, ChangeAmount: 100},
		{PlayerID: b.ID, ChangeAmount: -100},
	})
	if err != nil {
		t.Fatalf("SubmitRecords: %v", err)
	}
	if view.RoundNo != 1 || len(view.Records) != 2 {
		t.Fatalf("view = %+v", view)
	}

	players, _ := s.ListPlayers(room.ID)
	chips := map[string]int{}
	for _, p := range players {
		chips[p.Name] = p.CurrentChip
	}
	if chips["Alice"] != 1100 || chips["Bob"] != 900 {
		t.Fatalf("chips = %+v", chips)
	}
}

func TestSubmitRecordsClosedRoom(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("test")
	a, _ := s.AddPlayer(room.RoomCode, "Alice", 1000)
	b, _ := s.AddPlayer(room.RoomCode, "Bob", 1000)
	round, _ := s.CreateRound(room.RoomCode)
	_, _ = s.CloseRoom(room.RoomCode)

	_, err := s.SubmitRecords(round.ID, []models.RecordInput{
		{PlayerID: a.ID, ChangeAmount: 50},
		{PlayerID: b.ID, ChangeAmount: -50},
	})
	if err != ErrRoomClosed {
		t.Fatalf("err = %v, want ErrRoomClosed", err)
	}
}

func TestCreateRoundIncrements(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("test")
	r1, _ := s.CreateRound(room.RoomCode)
	r2, _ := s.CreateRound(room.RoomCode)
	if r1.RoundNo != 1 || r2.RoundNo != 2 {
		t.Fatalf("round nos = %d, %d", r1.RoundNo, r2.RoundNo)
	}
}

func TestCloseRoomAndHistory(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("归档测试")
	a, _ := s.AddPlayer(room.RoomCode, "Alice", 1000)
	b, _ := s.AddPlayer(room.RoomCode, "Bob", 1000)
	round, _ := s.CreateRound(room.RoomCode)
	_, _ = s.SubmitRecords(round.ID, []models.RecordInput{
		{PlayerID: a.ID, ChangeAmount: 200},
		{PlayerID: b.ID, ChangeAmount: -200},
	})
	closed, err := s.CloseRoom(room.RoomCode)
	if err != nil {
		t.Fatal(err)
	}
	if closed.Status != models.RoomStatusClosed {
		t.Fatalf("status = %s", closed.Status)
	}

	history, err := s.GetHistory(room.RoomCode)
	if err != nil {
		t.Fatal(err)
	}
	if history.TotalRounds != 1 {
		t.Fatalf("totalRounds = %d", history.TotalRounds)
	}
	if history.ClosedAt == nil {
		t.Fatal("expected closedAt")
	}
	if len(history.Stats) != 2 {
		t.Fatalf("stats len = %d", len(history.Stats))
	}

	var aliceRank int
	for _, st := range history.Stats {
		if st.Name == "Alice" {
			aliceRank = st.Rank
			if st.TotalChange != 200 {
				t.Fatalf("Alice totalChange = %d", st.TotalChange)
			}
		}
	}
	if aliceRank != 1 {
		t.Fatalf("Alice rank = %d, want 1", aliceRank)
	}
}

func TestGetStatsRanking(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("stats")
	a, _ := s.AddPlayer(room.RoomCode, "A", 100)
	b, _ := s.AddPlayer(room.RoomCode, "B", 100)
	c, _ := s.AddPlayer(room.RoomCode, "C", 100)
	round, _ := s.CreateRound(room.RoomCode)
	_, err := s.SubmitRecords(round.ID, []models.RecordInput{
		{PlayerID: a.ID, ChangeAmount: 50},
		{PlayerID: b.ID, ChangeAmount: -30},
		{PlayerID: c.ID, ChangeAmount: -20},
	})
	if err != nil {
		t.Fatal(err)
	}
	stats, err := s.GetStats(room.RoomCode)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]models.PlayerStat{}
	for _, p := range stats.Players {
		byName[p.Name] = p
	}
	if byName["A"].Rank != 1 || byName["C"].Rank != 2 || byName["B"].Rank != 3 {
		t.Fatalf("ranks: A=%d B=%d C=%d", byName["A"].Rank, byName["B"].Rank, byName["C"].Rank)
	}
}

func TestGetRoomDetail(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("detail")
	a, _ := s.AddPlayer(room.RoomCode, "Alice", 500)
	b, _ := s.AddPlayer(room.RoomCode, "Bob", 500)
	round, _ := s.CreateRound(room.RoomCode)
	_, _ = s.SubmitRecords(round.ID, []models.RecordInput{
		{PlayerID: a.ID, ChangeAmount: 10},
		{PlayerID: b.ID, ChangeAmount: -10},
	})

	detail, err := s.GetRoomDetail(room.RoomCode)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Players) != 2 || len(detail.Rounds) != 1 {
		t.Fatalf("detail players=%d rounds=%d", len(detail.Players), len(detail.Rounds))
	}
	if len(detail.Rounds[0].Records) != 2 {
		t.Fatalf("records = %d", len(detail.Rounds[0].Records))
	}
}

func TestGetRoomCodeByRoundID(t *testing.T) {
	s := tempStore(t)
	room, _ := s.CreateRoom("code")
	round, _ := s.CreateRound(room.RoomCode)
	code, err := s.GetRoomCodeByRoundID(round.ID)
	if err != nil {
		t.Fatal(err)
	}
	if code != room.RoomCode {
		t.Fatalf("code = %s, want %s", code, room.RoomCode)
	}
}
