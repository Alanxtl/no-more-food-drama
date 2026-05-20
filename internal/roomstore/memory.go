package roomstore

import (
	"context"
	"sync"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type MemoryStore struct {
	mu      sync.Mutex
	rooms   map[string]memoryEntry
	nowFunc func() time.Time
}

type memoryEntry struct {
	room      domain.Room
	expiresAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		rooms:   map[string]memoryEntry{},
		nowFunc: time.Now,
	}
}

func (s *MemoryStore) SetNow(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nowFunc = func() time.Time {
		return now
	}
}

func (s *MemoryStore) Save(ctx context.Context, room domain.Room, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.rooms[room.ID] = memoryEntry{
		room:      room,
		expiresAt: s.nowFunc().Add(ttl),
	}
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, roomID string) (domain.Room, error) {
	if err := ctx.Err(); err != nil {
		return domain.Room{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.getLocked(roomID)
}

func (s *MemoryStore) Update(ctx context.Context, roomID string, ttl time.Duration, mutate func(domain.Room) (domain.Room, error)) (domain.Room, error) {
	if err := ctx.Err(); err != nil {
		return domain.Room{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	room, err := s.getLocked(roomID)
	if err != nil {
		return domain.Room{}, err
	}
	updated, err := mutate(room)
	if err != nil {
		return domain.Room{}, err
	}
	s.rooms[roomID] = memoryEntry{
		room:      updated,
		expiresAt: s.nowFunc().Add(ttl),
	}
	return updated, nil
}

func (s *MemoryStore) getLocked(roomID string) (domain.Room, error) {
	entry, ok := s.rooms[roomID]
	if !ok {
		return domain.Room{}, ErrRoomNotFound
	}
	if s.nowFunc().After(entry.expiresAt) {
		delete(s.rooms, roomID)
		return domain.Room{}, ErrRoomExpired
	}
	return entry.room, nil
}
