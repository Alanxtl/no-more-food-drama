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

func TestMemoryStoreDoesNotLeakMutableRoomStateAfterSaveOrGet(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	store.SetNow(now)
	room, participantID := domain.NewRoom("ABC123", "https://app.test/room/ABC123", now)
	participant := room.Participants[participantID]
	participant.TypeVotes["type-hotpot"] = domain.VoteWant
	room.Participants[participantID] = participant
	room.Restaurants = append(room.Restaurants, domain.Restaurant{ID: "restaurant-1", Name: "First"})

	if err := store.Save(ctx, room, time.Hour); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	participant = room.Participants[participantID]
	participant.TypeVotes["type-hotpot"] = domain.VoteAvoid
	room.Participants[participantID] = participant
	room.Restaurants[0].Name = "Mutated outside store"

	got, err := store.Get(ctx, room.ID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.Participants[participantID].TypeVotes["type-hotpot"] != domain.VoteWant {
		t.Fatalf("stored type vote was mutated through original room: %q", got.Participants[participantID].TypeVotes["type-hotpot"])
	}
	if got.Restaurants[0].Name != "First" {
		t.Fatalf("stored restaurant name was mutated through original room: %q", got.Restaurants[0].Name)
	}

	participant = got.Participants[participantID]
	participant.TypeVotes["type-hotpot"] = domain.VoteNeutral
	got.Participants[participantID] = participant
	got.Restaurants[0].Name = "Mutated returned room"

	gotAgain, err := store.Get(ctx, room.ID)
	if err != nil {
		t.Fatalf("second Get returned error: %v", err)
	}
	if gotAgain.Participants[participantID].TypeVotes["type-hotpot"] != domain.VoteWant {
		t.Fatalf("stored type vote was mutated through Get result: %q", gotAgain.Participants[participantID].TypeVotes["type-hotpot"])
	}
	if gotAgain.Restaurants[0].Name != "First" {
		t.Fatalf("stored restaurant name was mutated through Get result: %q", gotAgain.Restaurants[0].Name)
	}
}
