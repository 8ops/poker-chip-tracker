package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"poker-chip-tracker/internal/models"
	"poker-chip-tracker/internal/store"
	"poker-chip-tracker/internal/ws"
)

func testServer(t *testing.T) (*Server, *http.ServeMux) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv := NewServer(st, ws.NewHub())
	mux := http.NewServeMux()
	srv.Register(mux)
	return srv, mux
}

func doJSON(t *testing.T, mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	return rr
}

func TestCreateRoomAPI(t *testing.T) {
	_, mux := testServer(t)
	rr := doJSON(t, mux, http.MethodPost, "/api/rooms", map[string]string{"name": "测试房"})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var resp models.CreateRoomResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.RoomCode == "" {
		t.Fatal("empty roomCode")
	}
}

func TestJoinRoomNotFoundAPI(t *testing.T) {
	_, mux := testServer(t)
	rr := doJSON(t, mux, http.MethodPost, "/api/rooms/ZZZZZZ/join", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestChipUnbalancedAPI(t *testing.T) {
	_, mux := testServer(t)
	rr := doJSON(t, mux, http.MethodPost, "/api/rooms", map[string]string{"name": "x"})
	var created models.CreateRoomResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &created)

	rr = doJSON(t, mux, http.MethodPost, "/api/players", map[string]any{
		"roomCode": created.RoomCode, "name": "A", "initialChip": 100,
	})
	var pA models.Player
	_ = json.Unmarshal(rr.Body.Bytes(), &pA)

	rr = doJSON(t, mux, http.MethodPost, "/api/players", map[string]any{
		"roomCode": created.RoomCode, "name": "B", "initialChip": 100,
	})
	var pB models.Player
	_ = json.Unmarshal(rr.Body.Bytes(), &pB)

	rr = doJSON(t, mux, http.MethodPost, "/api/rooms/"+created.RoomCode+"/rounds", nil)
	var round models.GameRound
	_ = json.Unmarshal(rr.Body.Bytes(), &round)

	rr = doJSON(t, mux, http.MethodPost, "/api/rounds/"+round.ID+"/records", map[string]any{
		"records": []map[string]any{
			{"playerId": pA.ID, "changeAmount": 10},
			{"playerId": pB.ID, "changeAmount": -5},
		},
	})
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}

	rr = doJSON(t, mux, http.MethodPost, "/api/rounds/"+round.ID+"/records", map[string]any{
		"records": []map[string]any{
			{"playerId": pA.ID, "changeAmount": 10},
			{"playerId": pB.ID, "changeAmount": -10},
		},
	})
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestClosedRoomRejectsRecordsAPI(t *testing.T) {
	_, mux := testServer(t)
	rr := doJSON(t, mux, http.MethodPost, "/api/rooms", map[string]string{"name": "x"})
	var created models.CreateRoomResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &created)

	rr = doJSON(t, mux, http.MethodPost, "/api/players", map[string]any{
		"roomCode": created.RoomCode, "name": "A", "initialChip": 100,
	})
	var pA models.Player
	_ = json.Unmarshal(rr.Body.Bytes(), &pA)
	rr = doJSON(t, mux, http.MethodPost, "/api/players", map[string]any{
		"roomCode": created.RoomCode, "name": "B", "initialChip": 100,
	})
	var pB models.Player
	_ = json.Unmarshal(rr.Body.Bytes(), &pB)

	rr = doJSON(t, mux, http.MethodPost, "/api/rooms/"+created.RoomCode+"/rounds", nil)
	var round models.GameRound
	_ = json.Unmarshal(rr.Body.Bytes(), &round)

	rr = doJSON(t, mux, http.MethodPost, "/api/rooms/"+created.RoomCode+"/close", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("close status = %d", rr.Code)
	}

	rr = doJSON(t, mux, http.MethodPost, "/api/rounds/"+round.ID+"/records", map[string]any{
		"records": []map[string]any{
			{"playerId": pA.ID, "changeAmount": 5},
			{"playerId": pB.ID, "changeAmount": -5},
		},
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}

	rr = doJSON(t, mux, http.MethodPost, "/api/players", map[string]any{
		"roomCode": created.RoomCode, "name": "C", "initialChip": 100,
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("add player status = %d", rr.Code)
	}
}

func TestFullFlowAPI(t *testing.T) {
	_, mux := testServer(t)
	rr := doJSON(t, mux, http.MethodPost, "/api/rooms", map[string]string{"name": "周末"})
	var created models.CreateRoomResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &created)

	rr = doJSON(t, mux, http.MethodPost, "/api/rooms/"+created.RoomCode+"/join", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("join = %d", rr.Code)
	}

	for _, name := range []string{"张三", "李四"} {
		rr = doJSON(t, mux, http.MethodPost, "/api/players", map[string]any{
			"roomCode": created.RoomCode, "name": name, "initialChip": 1000,
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("add %s: %d %s", name, rr.Code, rr.Body.String())
		}
	}

	rr = doJSON(t, mux, http.MethodGet, "/api/rooms/"+created.RoomCode+"/stats", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("stats = %d", rr.Code)
	}

	rr = doJSON(t, mux, http.MethodGet, "/api/rooms/"+created.RoomCode+"/history", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("history = %d", rr.Code)
	}
}
