package roomstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func TestMemoryStoreDoesNotReturnExpiredRoom(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	store.SetNow(now)
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)

	if err := store.Save(ctx, room, time.Hour); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	store.SetNow(now.Add(time.Hour + time.Nanosecond))

	_, err := store.Get(ctx, room.ID)
	if !errors.Is(err, ErrRoomExpired) {
		t.Fatalf("Get error = %v, want %v", err, ErrRoomExpired)
	}
	_, err = store.Get(ctx, room.ID)
	if !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("Get after expired deletion error = %v, want %v", err, ErrRoomNotFound)
	}
}

func TestMemoryStoreUpdateRefreshesRoomTTL(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	store.SetNow(now)
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)

	if err := store.Save(ctx, room, time.Hour); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	store.SetNow(now.Add(30 * time.Minute))
	updated, err := store.Update(ctx, room.ID, time.Hour, func(room domain.Room) (domain.Room, error) {
		room.Version++
		room.ShareURL = "https://app.test/room/ABC123?updated=1"
		return room, nil
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("updated version = %d, want 2", updated.Version)
	}

	store.SetNow(now.Add(time.Hour + time.Minute))
	got, err := store.Get(ctx, room.ID)
	if err != nil {
		t.Fatalf("Get after original expiry returned error: %v", err)
	}
	if got.ShareURL != "https://app.test/room/ABC123?updated=1" {
		t.Fatalf("ShareURL = %q", got.ShareURL)
	}
}
