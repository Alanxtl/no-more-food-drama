package roomstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestUpstashStoreSaveRefreshesExpiresAtAndRejectsNonOKResult(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now.Add(-time.Hour))
	var storedRoom domain.Room
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var command []any
		if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		if err := json.Unmarshal([]byte(command[2].(string)), &storedRoom); err != nil {
			t.Fatalf("Unmarshal stored room: %v", err)
		}
		_, _ = w.Write([]byte(`{"result":"QUEUED"}`))
	}))
	defer server.Close()

	store := NewUpstashStore(server.URL, "test-token", server.Client())
	store.nowFunc = func() time.Time { return now }
	err := store.Save(ctx, room, time.Hour)
	if err == nil {
		t.Fatal("Save returned nil error for non-OK SET result")
	}
	if got := storedRoom.ExpiresAt; !got.Equal(now.Add(time.Hour)) {
		t.Fatalf("stored ExpiresAt = %s, want %s", got, now.Add(time.Hour))
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

func TestUpstashStoreGetMissingResultFieldIsProtocolError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	store := NewUpstashStore(server.URL, "test-token", server.Client())
	_, err := store.Get(ctx, "missing")
	if err == nil {
		t.Fatal("Get returned nil error for missing result field")
	}
	if errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("Get error = %v, want protocol error not %v", err, ErrRoomNotFound)
	}
}

func TestUpstashStoreUpdateRefreshesExpiresAtAndUsesVersionAwareCommand(t *testing.T) {
	ctx := context.Background()
	readAt := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	updateAt := readAt.Add(10 * time.Minute)
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", readAt)
	roomJSON, err := json.Marshal(room)
	if err != nil {
		t.Fatalf("Marshal room: %v", err)
	}

	var evalCommand []any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var command []any
		if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		switch command[0] {
		case "GET":
			_ = json.NewEncoder(w).Encode(map[string]any{"result": string(roomJSON)})
		case "EVAL":
			evalCommand = command
			_, _ = w.Write([]byte(`{"result":"OK"}`))
		default:
			t.Fatalf("unexpected command = %#v", command)
		}
	}))
	defer server.Close()

	store := NewUpstashStore(server.URL+"/", "test-token", server.Client())
	store.nowFunc = func() time.Time { return updateAt }
	updated, err := store.Update(ctx, room.ID, time.Hour, func(room domain.Room) (domain.Room, error) {
		room.Version++
		room.ShareURL = "https://app.test/room/ABC123?updated=1"
		return room, nil
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !updated.ExpiresAt.Equal(updateAt.Add(time.Hour)) {
		t.Fatalf("updated ExpiresAt = %s, want %s", updated.ExpiresAt, updateAt.Add(time.Hour))
	}
	if len(evalCommand) != 7 {
		t.Fatalf("EVAL command length = %d, want 7: %#v", len(evalCommand), evalCommand)
	}
	if evalCommand[0] != "EVAL" || evalCommand[2] != float64(1) || evalCommand[3] != "room:ABC123" {
		t.Fatalf("EVAL command = %#v", evalCommand)
	}
	if !strings.Contains(evalCommand[1].(string), "version") {
		t.Fatalf("EVAL script does not compare version: %q", evalCommand[1])
	}
	if evalCommand[4] != float64(1) {
		t.Fatalf("expected version argument = %#v, want 1", evalCommand[4])
	}
	storedJSON, ok := evalCommand[5].(string)
	if !ok {
		t.Fatalf("stored value type = %T, want string", evalCommand[5])
	}
	var stored domain.Room
	if err := json.Unmarshal([]byte(storedJSON), &stored); err != nil {
		t.Fatalf("Unmarshal CAS stored room: %v", err)
	}
	if stored.Version != 2 {
		t.Fatalf("stored version = %d, want 2", stored.Version)
	}
	if !stored.ExpiresAt.Equal(updateAt.Add(time.Hour)) {
		t.Fatalf("stored ExpiresAt = %s, want %s", stored.ExpiresAt, updateAt.Add(time.Hour))
	}
	if evalCommand[6] != float64(3600) {
		t.Fatalf("ttl seconds = %#v, want 3600", evalCommand[6])
	}
}

func TestUpstashStoreUpdateRetriesVersionConflict(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	firstRoom, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)
	secondRoom := firstRoom
	secondRoom.Version = 2
	rooms := []domain.Room{firstRoom, secondRoom}
	gets := 0
	evals := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var command []any
		if err := json.NewDecoder(r.Body).Decode(&command); err != nil {
			t.Fatalf("Decode request body: %v", err)
		}
		switch command[0] {
		case "GET":
			if gets >= len(rooms) {
				t.Fatalf("too many GET requests")
			}
			roomJSON, err := json.Marshal(rooms[gets])
			if err != nil {
				t.Fatalf("Marshal room %d: %v", gets, err)
			}
			gets++
			_ = json.NewEncoder(w).Encode(map[string]any{"result": string(roomJSON)})
		case "EVAL":
			evals++
			if evals == 1 {
				_, _ = w.Write([]byte(`{"result":"VERSION_CONFLICT"}`))
				return
			}
			_, _ = w.Write([]byte(`{"result":"OK"}`))
		default:
			t.Fatalf("unexpected command = %#v", command)
		}
	}))
	defer server.Close()

	store := NewUpstashStore(server.URL, "test-token", server.Client())
	store.nowFunc = func() time.Time { return now }
	updated, err := store.Update(ctx, firstRoom.ID, time.Hour, func(room domain.Room) (domain.Room, error) {
		room.Version++
		room.ShareURL = fmt.Sprintf("https://app.test/room/ABC123?v=%d", room.Version)
		return room, nil
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if gets != 2 {
		t.Fatalf("GET count = %d, want 2", gets)
	}
	if evals != 2 {
		t.Fatalf("EVAL count = %d, want 2", evals)
	}
	if updated.Version != 3 {
		t.Fatalf("updated version = %d, want 3 after retry", updated.Version)
	}
}
