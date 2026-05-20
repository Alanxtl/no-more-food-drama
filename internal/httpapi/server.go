package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/recommend"
	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
	"github.com/Alanxtl/no-more-food-drama/internal/tagging"
)

type RestaurantProvider interface {
	SearchAround(ctx context.Context, lat float64, lng float64, radiusKM int, limit int) ([]domain.Restaurant, error)
}

type Config struct {
	AppURL      string
	Store       roomstore.Store
	Restaurants RestaurantProvider
}

type Server struct {
	config Config
	now    func() time.Time
}

func NewServer(config Config) *Server {
	return &Server{config: config, now: time.Now}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path, ok := strings.CutPrefix(r.URL.Path, "/api/rooms")
	if !ok {
		writeFailure(w, http.StatusNotFound, domain.ErrorValidation, "unknown route")
		return
	}
	if path == "" && r.Method == http.MethodPost {
		s.createRoom(w, r)
		return
	}

	parts := splitPath(path)
	if len(parts) == 1 && r.Method == http.MethodGet {
		s.snapshot(w, r, parts[0])
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		switch parts[1] {
		case "join":
			s.joinRoom(w, r, parts[0])
		case "search":
			s.search(w, r, parts[0])
		case "recommendations":
			s.recommendations(w, r, parts[0])
		default:
			writeFailure(w, http.StatusNotFound, domain.ErrorValidation, "unknown route")
		}
		return
	}

	writeFailure(w, http.StatusNotFound, domain.ErrorValidation, "unknown route")
}

func (s *Server) createRoom(w http.ResponseWriter, r *http.Request) {
	roomID, err := randomRoomID()
	if err != nil {
		writeFailure(w, http.StatusInternalServerError, domain.ErrorProvider, "room creation failed")
		return
	}

	shareURL := strings.TrimRight(s.config.AppURL, "/") + "/room/" + roomID
	room, participantID := domain.NewRoom(roomID, shareURL, s.now())
	if err := s.config.Store.Save(r.Context(), room, domain.RoomTTL); err != nil {
		writeFailure(w, http.StatusInternalServerError, domain.ErrorProvider, "room creation failed")
		return
	}

	writeSuccess(w, map[string]any{
		"roomId":        room.ID,
		"participantId": participantID,
		"shareUrl":      shareURL,
		"room":          room,
	})
}

func (s *Server) snapshot(w http.ResponseWriter, r *http.Request, roomID string) {
	room, err := s.config.Store.Get(r.Context(), roomID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}

func (s *Server) joinRoom(w http.ResponseWriter, r *http.Request, roomID string) {
	var participantID string
	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		participantID = room.JoinPartner(s.now())
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"participantId": participantID, "room": room})
}

func (s *Server) search(w http.ResponseWriter, r *http.Request, roomID string) {
	var input struct {
		Lat      float64 `json:"lat"`
		Lng      float64 `json:"lng"`
		RadiusKM int     `json:"radiusKm"`
		Limit    int     `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeFailure(w, http.StatusBadRequest, domain.ErrorValidation, "invalid search request")
		return
	}

	restaurants, err := s.config.Restaurants.SearchAround(r.Context(), input.Lat, input.Lng, input.RadiusKM, input.Limit)
	if err != nil {
		writeFailure(w, http.StatusBadGateway, domain.ErrorProvider, "restaurant search failed")
		return
	}
	tagged, types := tagging.BuildRuleTags(restaurants)

	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		room.SearchConfig = &domain.SearchConfig{
			Lat:      input.Lat,
			Lng:      input.Lng,
			RadiusKM: input.RadiusKM,
			Limit:    input.Limit,
		}
		room.Restaurants = tagged
		room.Types = types
		room.Recommendations = []domain.Recommendation{}
		room.Status = domain.StatusFiltering
		room.Version++
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}

func (s *Server) recommendations(w http.ResponseWriter, r *http.Request, roomID string) {
	room, err := s.config.Store.Update(r.Context(), roomID, domain.RoomTTL, func(room domain.Room) (domain.Room, error) {
		room.Recommendations = recommend.Compute(room, 5)
		room.Status = domain.StatusResults
		room.Version++
		return room, nil
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeSuccess(w, map[string]any{"room": room})
}

func writeSuccess(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(domain.Success(data))
}

func writeFailure(w http.ResponseWriter, status int, code string, message string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(domain.Failure(code, message))
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, roomstore.ErrRoomExpired):
		writeFailure(w, http.StatusGone, domain.ErrorRoomExpired, "room expired")
	case errors.Is(err, roomstore.ErrRoomNotFound):
		writeFailure(w, http.StatusNotFound, domain.ErrorRoomNotFound, "room not found")
	default:
		writeFailure(w, http.StatusInternalServerError, domain.ErrorProvider, "room update failed")
	}
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func randomRoomID() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var out strings.Builder
	out.Grow(6)
	for range 6 {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		out.WriteByte(alphabet[n.Int64()])
	}
	return out.String(), nil
}
