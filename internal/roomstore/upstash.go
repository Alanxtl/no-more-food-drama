package roomstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type UpstashStore struct {
	baseURL string
	token   string
	client  *http.Client
}

type upstashResponse struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

func NewUpstashStore(baseURL, token string, client *http.Client) *UpstashStore {
	if client == nil {
		client = http.DefaultClient
	}
	return &UpstashStore{
		baseURL: baseURL,
		token:   token,
		client:  client,
	}
}

func (s *UpstashStore) Save(ctx context.Context, room domain.Room, ttl time.Duration) error {
	value, err := json.Marshal(room)
	if err != nil {
		return err
	}
	command := []any{"SET", roomKey(room.ID), string(value), "EX", int(ttl.Seconds())}
	_, err = s.do(ctx, command)
	return err
}

func (s *UpstashStore) Get(ctx context.Context, roomID string) (domain.Room, error) {
	resp, err := s.do(ctx, []any{"GET", roomKey(roomID)})
	if err != nil {
		return domain.Room{}, err
	}
	if len(resp.Result) == 0 || bytes.Equal(resp.Result, []byte("null")) {
		return domain.Room{}, ErrRoomNotFound
	}

	var roomJSON string
	if err := json.Unmarshal(resp.Result, &roomJSON); err != nil {
		return domain.Room{}, err
	}
	var room domain.Room
	if err := json.Unmarshal([]byte(roomJSON), &room); err != nil {
		return domain.Room{}, err
	}
	if time.Now().After(room.ExpiresAt) {
		return domain.Room{}, ErrRoomExpired
	}
	return room, nil
}

func (s *UpstashStore) Update(ctx context.Context, roomID string, ttl time.Duration, mutate func(domain.Room) (domain.Room, error)) (domain.Room, error) {
	room, err := s.Get(ctx, roomID)
	if err != nil {
		return domain.Room{}, err
	}
	updated, err := mutate(room)
	if err != nil {
		return domain.Room{}, err
	}
	if err := s.Save(ctx, updated, ttl); err != nil {
		return domain.Room{}, err
	}
	return updated, nil
}

func (s *UpstashStore) do(ctx context.Context, command []any) (upstashResponse, error) {
	body, err := json.Marshal(command)
	if err != nil {
		return upstashResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(body))
	if err != nil {
		return upstashResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	httpResp, err := s.client.Do(req)
	if err != nil {
		return upstashResponse{}, err
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return upstashResponse{}, err
	}
	if httpResp.StatusCode >= http.StatusBadRequest {
		return upstashResponse{}, fmt.Errorf("upstash returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var resp upstashResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return upstashResponse{}, err
	}
	if resp.Error != "" {
		return upstashResponse{}, fmt.Errorf("upstash error: %s", resp.Error)
	}
	return resp, nil
}
