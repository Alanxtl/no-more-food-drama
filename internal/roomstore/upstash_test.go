package roomstore

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func TestUpstashStoreSendsSetWithOneHourExpiry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)

	var gotAuth string
	var gotContentType string
	var command []any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		_, _ = w.Write([]byte(`{"result":"OK"}`))
	}))
	defer server.Close()

	store := NewUpstashStore(server.URL, "test-token", server.Client())
	if err := store.Save(ctx, room, time.Hour); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if gotAuth != "Bearer test-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if len(command) != 5 {
		t.Fatalf("command length = %d, want 5: %#v", len(command), command)
	}
	if command[0] != "SET" || command[1] != "room:ABC123" || command[3] != "EX" {
		t.Fatalf("command = %#v", command)
	}
	if command[4] != float64(3600) {
		t.Fatalf("expiry seconds = %#v, want 3600", command[4])
	}
	storedRoomJSON, ok := command[2].(string)
	if !ok {
		t.Fatalf("stored value type = %T, want string", command[2])
	}
	var storedRoom domain.Room
	if err := json.Unmarshal([]byte(storedRoomJSON), &storedRoom); err != nil {
		t.Fatalf("stored room JSON did not unmarshal: %v", err)
	}
	if storedRoom.ID != room.ID {
		t.Fatalf("stored room ID = %q, want %q", storedRoom.ID, room.ID)
	}
}

func TestUpstashStoreGetDecodesRoom(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)
	roomJSON, err := json.Marshal(room)
	if err != nil {
		t.Fatalf("Marshal room: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var command []any
		if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		if command[0] != "GET" || command[1] != "room:ABC123" {
			t.Fatalf("command = %#v", command)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"result": string(roomJSON)})
	}))
	defer server.Close()

	store := NewUpstashStore(server.URL, "test-token", server.Client())
	got, err := store.Get(ctx, room.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ID != room.ID {
		t.Fatalf("room ID = %q, want %q", got.ID, room.ID)
	}
}

func TestUpstashStoreGetMissingRoom(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":null}`))
	}))
	defer server.Close()

	store := NewUpstashStore(server.URL, "test-token", server.Client())
	_, err := store.Get(ctx, "missing")
	if !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("Get error = %v, want %v", err, ErrRoomNotFound)
	}
}
