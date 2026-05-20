package roomstore

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrRoomExpired  = errors.New("room expired")
)

type Store interface {
	Save(ctx context.Context, room domain.Room, ttl time.Duration) error
	Get(ctx context.Context, roomID string) (domain.Room, error)
	Update(ctx context.Context, roomID string, ttl time.Duration, mutate func(domain.Room) (domain.Room, error)) (domain.Room, error)
}

func roomKey(roomID string) string {
	return "room:" + roomID
}

func cloneRoom(room domain.Room) (domain.Room, error) {
	data, err := json.Marshal(room)
	if err != nil {
		return domain.Room{}, err
	}
	var cloned domain.Room
	if err := json.Unmarshal(data, &cloned); err != nil {
		return domain.Room{}, err
	}
	return cloned, nil
}
