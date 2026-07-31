package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"poker-chip-tracker/internal/models"
	"poker-chip-tracker/internal/store"
	"poker-chip-tracker/internal/ws"
)

type Server struct {
	store *store.Store
	hub   *ws.Hub
}

func NewServer(st *store.Store, hub *ws.Hub) *Server {
	return &Server{store: st, hub: hub}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/rooms", s.createRoom)
	mux.HandleFunc("POST /api/rooms/{code}/join", s.joinRoom)
	mux.HandleFunc("GET /api/rooms/{code}", s.getRoom)
	mux.HandleFunc("POST /api/rooms/{code}/close", s.closeRoom)
	mux.HandleFunc("GET /api/rooms/{code}/stats", s.getStats)
	mux.HandleFunc("GET /api/rooms/{code}/history", s.getHistory)
	mux.HandleFunc("POST /api/players", s.addPlayer)
	mux.HandleFunc("POST /api/rooms/{code}/rounds", s.createRound)
	mux.HandleFunc("POST /api/rounds/{id}/records", s.submitRecords)
	mux.HandleFunc("GET /ws/rooms/{code}", s.handleWS)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(r.PathValue("code"))
	if _, err := s.store.GetRoomByCode(code); err != nil {
		writeError(w, http.StatusNotFound, "room not found")
		return
	}
	s.hub.HandleWS(w, r, code)
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRoomRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	room, err := s.store.CreateRoom(req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create room")
		return
	}
	writeJSON(w, http.StatusOK, models.CreateRoomResponse{RoomCode: room.RoomCode, RoomID: room.ID})
}

func (s *Server) joinRoom(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	room, err := s.store.GetRoomByCode(code)
	if err != nil {
		if errors.Is(err, store.ErrRoomNotFound) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	detail, err := s.store.GetRoomDetail(room.RoomCode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) getRoom(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	detail, err := s.store.GetRoomDetail(code)
	if err != nil {
		if errors.Is(err, store.ErrRoomNotFound) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) closeRoom(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	room, err := s.store.CloseRoom(code)
	if err != nil {
		if errors.Is(err, store.ErrRoomNotFound) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	s.hub.Broadcast(room.RoomCode, "room_closed", room)
	writeJSON(w, http.StatusOK, room)
}

func (s *Server) addPlayer(w http.ResponseWriter, r *http.Request) {
	var req models.AddPlayerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	player, err := s.store.AddPlayer(req.RoomCode, req.Name, req.InitialChip)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRoomNotFound):
			writeError(w, http.StatusNotFound, "room not found")
		case errors.Is(err, store.ErrRoomClosed):
			writeError(w, http.StatusForbidden, "room is closed")
		case errors.Is(err, store.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "server error")
		}
		return
	}
	s.hub.Broadcast(strings.ToUpper(strings.TrimSpace(req.RoomCode)), "player_added", player)
	writeJSON(w, http.StatusOK, player)
}

func (s *Server) createRound(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	round, err := s.store.CreateRound(code)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRoomNotFound):
			writeError(w, http.StatusNotFound, "room not found")
		case errors.Is(err, store.ErrRoomClosed):
			writeError(w, http.StatusForbidden, "room is closed")
		default:
			writeError(w, http.StatusInternalServerError, "server error")
		}
		return
	}
	room, _ := s.store.GetRoomByCode(code)
	if room != nil {
		s.hub.Broadcast(room.RoomCode, "round_created", round)
	}
	writeJSON(w, http.StatusOK, round)
}

func (s *Server) submitRecords(w http.ResponseWriter, r *http.Request) {
	roundID := r.PathValue("id")
	var req models.SubmitRecordsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	roundView, err := s.store.SubmitRecords(roundID, req.Records)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrChipUnbalanced):
			writeError(w, http.StatusConflict, "筹码未平衡")
		case errors.Is(err, store.ErrRoomClosed):
			writeError(w, http.StatusForbidden, "room is closed")
		case errors.Is(err, store.ErrPlayerNotFound):
			writeError(w, http.StatusBadRequest, "player not found")
		case errors.Is(err, store.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, roundView)

	if roomCode, err := s.store.GetRoomCodeByRoundID(roundID); err == nil {
		if detail, err := s.store.GetRoomDetail(roomCode); err == nil {
			s.hub.Broadcast(roomCode, "records_submitted", detail)
		}
	}
}

func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	stats, err := s.store.GetStats(code)
	if err != nil {
		if errors.Is(err, store.ErrRoomNotFound) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) getHistory(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	history, err := s.store.GetHistory(code)
	if err != nil {
		if errors.Is(err, store.ErrRoomNotFound) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, models.ErrorResponse{Code: status, Message: message})
}
