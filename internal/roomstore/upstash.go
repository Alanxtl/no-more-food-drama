package roomstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type UpstashStore struct {
	baseURL string
	token   string
	client  *http.Client
	nowFunc func() time.Time
}

type upstashResponse struct {
	Result    json.RawMessage
	HasResult bool
	Error     string
}

var errVersionConflict = errors.New("room version conflict")

const (
	upstashOK          = "OK"
	upstashConflict    = "VERSION_CONFLICT"
	upstashUpdateTries = 3
	upstashCASScript   = `
local current = redis.call("GET", KEYS[1])
if not current then
  return nil
end
local room = cjson.decode(current)
if tonumber(room["version"]) ~= tonumber(ARGV[1]) then
  return "VERSION_CONFLICT"
end
redis.call("SET", KEYS[1], ARGV[2], "EX", ARGV[3])
return "OK"
`
)

func NewUpstashStore(baseURL, token string, client *http.Client) *UpstashStore {
	if client == nil {
		client = http.DefaultClient
	}
	return &UpstashStore{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  client,
		nowFunc: time.Now,
	}
}

func (s *UpstashStore) Save(ctx context.Context, room domain.Room, ttl time.Duration) error {
	room.ExpiresAt = s.nowFunc().Add(ttl)
	value, err := json.Marshal(room)
	if err != nil {
		return err
	}
	command := []any{"SET", roomKey(room.ID), string(value), "EX", int(ttl.Seconds())}
	resp, err := s.do(ctx, command)
	if err != nil {
		return err
	}
	result, err := responseString(resp)
	if err != nil {
		return err
	}
	if result != upstashOK {
		return fmt.Errorf("upstash SET result = %q, want %q", result, upstashOK)
	}
	return nil
}

func (s *UpstashStore) Get(ctx context.Context, roomID string) (domain.Room, error) {
	resp, err := s.do(ctx, []any{"GET", roomKey(roomID)})
	if err != nil {
		return domain.Room{}, err
	}
	if !resp.HasResult {
		return domain.Room{}, errors.New("upstash response missing result field")
	}
	if bytes.Equal(resp.Result, []byte("null")) {
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
	if s.nowFunc().After(room.ExpiresAt) {
		return domain.Room{}, ErrRoomExpired
	}
	return room, nil
}

func (s *UpstashStore) Update(ctx context.Context, roomID string, ttl time.Duration, mutate func(domain.Room) (domain.Room, error)) (domain.Room, error) {
	var lastConflict error
	for range upstashUpdateTries {
		room, err := s.Get(ctx, roomID)
		if err != nil {
			return domain.Room{}, err
		}
		expectedVersion := room.Version
		updated, err := mutate(room)
		if err != nil {
			return domain.Room{}, err
		}
		updated.ExpiresAt = s.nowFunc().Add(ttl)
		if err := s.compareAndSet(ctx, roomID, expectedVersion, updated, ttl); err != nil {
			if errors.Is(err, errVersionConflict) {
				lastConflict = err
				continue
			}
			return domain.Room{}, err
		}
		return updated, nil
	}
	return domain.Room{}, lastConflict
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

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return upstashResponse{}, err
	}
	if rawError, ok := raw["error"]; ok && !bytes.Equal(rawError, []byte("null")) {
		var upstashErr string
		if err := json.Unmarshal(rawError, &upstashErr); err != nil {
			return upstashResponse{}, fmt.Errorf("upstash error: %s", string(rawError))
		}
		if upstashErr != "" {
			return upstashResponse{}, fmt.Errorf("upstash error: %s", upstashErr)
		}
	}
	result, hasResult := raw["result"]
	resp := upstashResponse{
		Result:    result,
		HasResult: hasResult,
	}
	return resp, nil
}

func (s *UpstashStore) compareAndSet(ctx context.Context, roomID string, expectedVersion int, room domain.Room, ttl time.Duration) error {
	value, err := json.Marshal(room)
	if err != nil {
		return err
	}
	command := []any{
		"EVAL",
		upstashCASScript,
		1,
		roomKey(roomID),
		expectedVersion,
		string(value),
		int(ttl.Seconds()),
	}
	resp, err := s.do(ctx, command)
	if err != nil {
		return err
	}
	if !resp.HasResult {
		return errors.New("upstash response missing result field")
	}
	if bytes.Equal(resp.Result, []byte("null")) {
		return ErrRoomNotFound
	}
	result, err := responseString(resp)
	if err != nil {
		return err
	}
	switch result {
	case upstashOK:
		return nil
	case upstashConflict:
		return errVersionConflict
	default:
		return fmt.Errorf("upstash CAS result = %q, want %q", result, upstashOK)
	}
}

func responseString(resp upstashResponse) (string, error) {
	if !resp.HasResult {
		return "", errors.New("upstash response missing result field")
	}
	var result string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", err
	}
	return result, nil
}
