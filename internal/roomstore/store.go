package roomstore

import (
	"context"
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
