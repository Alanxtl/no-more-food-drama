package domain

import (
	"testing"
	"time"
)

func TestNewRoomCreatesCreatorAndOneHourExpiry(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, participantID := NewRoom("ABC123", "https://app.test/room/ABC123", now)

	if room.ID != "ABC123" {
		t.Fatalf("room id = %q", room.ID)
	}
	if participantID == "" {
		t.Fatal("participant id is empty")
	}
	participant := room.Participants[participantID]
	if participant.Role != RoleCreator {
		t.Fatalf("role = %q", participant.Role)
	}
	if got := room.ExpiresAt.Sub(now); got != time.Hour {
		t.Fatalf("expiry duration = %s", got)
	}
	if room.Status != StatusLobby {
		t.Fatalf("status = %q", room.Status)
	}
}

func TestTypeVoteIsSoftPreference(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, participantID := NewRoom("ABC123", "https://app.test/room/ABC123", now)

	err := room.SetTypeVote(participantID, "type-japanese", VoteAvoid, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("SetTypeVote returned error: %v", err)
	}

	if got := room.Participants[participantID].TypeVotes["type-japanese"]; got != VoteAvoid {
		t.Fatalf("vote = %q", got)
	}
	if room.Version != 2 {
		t.Fatalf("version = %d", room.Version)
	}
	if got := room.ExpiresAt.Sub(now.Add(time.Minute)); got != time.Hour {
		t.Fatalf("renewed ttl = %s", got)
	}
}

func TestRestaurantRemoveOverrideIsStored(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, participantID := NewRoom("ABC123", "https://app.test/room/ABC123", now)

	err := room.SetRestaurantOverride(participantID, "restaurant-1", RestaurantRemove, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("SetRestaurantOverride returned error: %v", err)
	}

	if got := room.Participants[participantID].RestaurantOverrides["restaurant-1"]; got != RestaurantRemove {
		t.Fatalf("override = %q", got)
	}
}

func TestUnknownParticipantReturnsError(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	room, _ := NewRoom("ABC123", "https://app.test/room/ABC123", now)

	err := room.SetTypeVote("missing", "type-hotpot", VoteWant, now)
	if err != ErrParticipantNotFound {
		t.Fatalf("error = %v", err)
	}
}
