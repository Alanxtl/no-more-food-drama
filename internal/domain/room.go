package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"
)

var ErrParticipantNotFound = errors.New("participant not found")

func NewRoom(id string, shareURL string, now time.Time) (Room, string) {
	participantID := newID("p")
	participant := Participant{
		DisplayName:         "我",
		Role:                RoleCreator,
		JoinedAt:            now,
		LastSeenAt:          now,
		TypeVotes:           map[string]TypeVote{},
		RestaurantOverrides: map[string]RestaurantOverride{},
	}

	return Room{
		ID:              id,
		Version:         1,
		ShareURL:        shareURL,
		CreatedAt:       now,
		ExpiresAt:       now.Add(RoomTTL),
		Status:          StatusLobby,
		Participants:    map[string]Participant{participantID: participant},
		Restaurants:     []Restaurant{},
		Types:           []FoodType{},
		Recommendations: []Recommendation{},
	}, participantID
}

func (r *Room) JoinPartner(now time.Time) string {
	participantID := newID("p")
	r.Participants[participantID] = Participant{
		DisplayName:         "另一位",
		Role:                RolePartner,
		JoinedAt:            now,
		LastSeenAt:          now,
		TypeVotes:           map[string]TypeVote{},
		RestaurantOverrides: map[string]RestaurantOverride{},
	}
	r.touch(now)
	return participantID
}

func (r *Room) Heartbeat(participantID string, now time.Time) error {
	p, ok := r.Participants[participantID]
	if !ok {
		return ErrParticipantNotFound
	}
	p.LastSeenAt = now
	r.Participants[participantID] = p
	r.touch(now)
	return nil
}

func (r *Room) SetTypeVote(participantID string, typeID string, vote TypeVote, now time.Time) error {
	p, ok := r.Participants[participantID]
	if !ok {
		return ErrParticipantNotFound
	}
	p.TypeVotes[typeID] = vote
	p.LastSeenAt = now
	r.Participants[participantID] = p
	r.touch(now)
	return nil
}

func (r *Room) SetRestaurantOverride(participantID string, restaurantID string, override RestaurantOverride, now time.Time) error {
	p, ok := r.Participants[participantID]
	if !ok {
		return ErrParticipantNotFound
	}
	p.RestaurantOverrides[restaurantID] = override
	p.LastSeenAt = now
	r.Participants[participantID] = p
	r.touch(now)
	return nil
}

func (r *Room) touch(now time.Time) {
	r.Version++
	r.ExpiresAt = now.Add(RoomTTL)
}

func newID(prefix string) string {
	var b [6]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
