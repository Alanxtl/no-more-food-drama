package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
)

func TestCreateAndJoinRoom(t *testing.T) {
	server := NewServer(Config{
		AppURL:      "https://app.test",
		Store:       roomstore.NewMemoryStore(),
		Restaurants: FakeRestaurantProvider{},
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms", nil)
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d body = %s", createRec.Code, createRec.Body.String())
	}
	var createBody envelope
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	roomID := createBody.Data["roomId"].(string)

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/join", nil)
	joinRec := httptest.NewRecorder()
	server.ServeHTTP(joinRec, joinReq)
	if joinRec.Code != http.StatusOK {
		t.Fatalf("join status = %d body = %s", joinRec.Code, joinRec.Body.String())
	}
}

func TestSearchWritesRuleTaggedRestaurants(t *testing.T) {
	server := NewServer(Config{
		AppURL:      "https://app.test",
		Store:       roomstore.NewMemoryStore(),
		Restaurants: FakeRestaurantProvider{},
	})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)

	body := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", body)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("search status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("type-japanese")) {
		t.Fatalf("search body missing type-japanese: %s", rec.Body.String())
	}
}

func TestSearchMissingRoomReturnsNotFoundWithoutCallingProvider(t *testing.T) {
	provider := &recordingRestaurantProvider{}
	server := NewServer(Config{
		AppURL:      "https://app.test",
		Store:       roomstore.NewMemoryStore(),
		Restaurants: provider,
	})

	body := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/MISSING/search", body)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("search status = %d body = %s", rec.Code, rec.Body.String())
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls)
	}
	assertErrorCode(t, rec.Body.Bytes(), domain.ErrorRoomNotFound)
}

func TestTypeVoteEndpointUpdatesParticipantVote(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)
	participantID := createBody.Data["participantId"].(string)

	body := bytes.NewBufferString(`{"participantId":"` + participantID + `","typeId":"type-hotpot","vote":"avoid"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/votes/type", body)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("vote status = %d body = %s", rec.Code, rec.Body.String())
	}
	room := decodeRoomResponse(t, rec.Body.Bytes())
	if got := room.Participants[participantID].TypeVotes["type-hotpot"]; got != domain.VoteAvoid {
		t.Fatalf("type vote = %q, want %q; body = %s", got, domain.VoteAvoid, rec.Body.String())
	}
}

func TestRestaurantOverrideEndpointUpdatesHardRemove(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)
	participantID := createBody.Data["participantId"].(string)

	body := bytes.NewBufferString(`{"participantId":"` + participantID + `","restaurantId":"amap:test-hotpot","override":"remove"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/votes/restaurant", body)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("override status = %d body = %s", rec.Code, rec.Body.String())
	}
	room := decodeRoomResponse(t, rec.Body.Bytes())
	if got := room.Participants[participantID].RestaurantOverrides["amap:test-hotpot"]; got != domain.RestaurantRemove {
		t.Fatalf("restaurant override = %q, want %q; body = %s", got, domain.RestaurantRemove, rec.Body.String())
	}
}

func TestVoteEndpointsValidatePayloads(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)
	participantID := createBody.Data["participantId"].(string)

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "invalid type json", path: "/api/rooms/" + roomID + "/votes/type", body: `{"participantId":`},
		{name: "missing type participant", path: "/api/rooms/" + roomID + "/votes/type", body: `{"typeId":"type-hotpot","vote":"avoid"}`},
		{name: "missing type id", path: "/api/rooms/" + roomID + "/votes/type", body: `{"participantId":"` + participantID + `","vote":"avoid"}`},
		{name: "missing type vote", path: "/api/rooms/" + roomID + "/votes/type", body: `{"participantId":"` + participantID + `","typeId":"type-hotpot"}`},
		{name: "invalid type vote", path: "/api/rooms/" + roomID + "/votes/type", body: `{"participantId":"` + participantID + `","typeId":"type-hotpot","vote":"maybe"}`},
		{name: "invalid restaurant json", path: "/api/rooms/" + roomID + "/votes/restaurant", body: `{"participantId":`},
		{name: "missing restaurant participant", path: "/api/rooms/" + roomID + "/votes/restaurant", body: `{"restaurantId":"amap:test-hotpot","override":"remove"}`},
		{name: "missing restaurant id", path: "/api/rooms/" + roomID + "/votes/restaurant", body: `{"participantId":"` + participantID + `","override":"remove"}`},
		{name: "missing restaurant override", path: "/api/rooms/" + roomID + "/votes/restaurant", body: `{"participantId":"` + participantID + `","restaurantId":"amap:test-hotpot"}`},
		{name: "invalid restaurant override", path: "/api/rooms/" + roomID + "/votes/restaurant", body: `{"participantId":"` + participantID + `","restaurantId":"amap:test-hotpot","override":"bananas"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.body)))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
			}
			assertErrorCode(t, rec.Body.Bytes(), domain.ErrorValidation)
		})
	}
}

func TestVoteEndpointUnknownParticipantReturnsFailureEnvelope(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "type vote", path: "/api/rooms/" + roomID + "/votes/type", body: `{"participantId":"missing","typeId":"type-hotpot","vote":"want"}`},
		{name: "restaurant override", path: "/api/rooms/" + roomID + "/votes/restaurant", body: `{"participantId":"missing","restaurantId":"amap:test-hotpot","override":"remove"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.body)))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
			}
			assertErrorCode(t, rec.Body.Bytes(), domain.ErrorParticipantNotFound)
		})
	}
}

func TestRewrittenSubroutePathJoinsExistingRoom(t *testing.T) {
	server := NewServer(Config{
		AppURL:      "https://app.test",
		Store:       roomstore.NewMemoryStore(),
		Restaurants: FakeRestaurantProvider{},
	})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	roomID := createBody.Data["roomId"].(string)

	joinReq := httptest.NewRequest(http.MethodPost, "/api/rooms?path="+roomID+"/join", nil)
	joinRec := httptest.NewRecorder()
	server.ServeHTTP(joinRec, joinReq)

	if joinRec.Code != http.StatusOK {
		t.Fatalf("join status = %d body = %s", joinRec.Code, joinRec.Body.String())
	}
	var joinBody envelope
	if err := json.Unmarshal(joinRec.Body.Bytes(), &joinBody); err != nil {
		t.Fatalf("decode join: %v", err)
	}
	if joinBody.Data["roomId"] != nil {
		t.Fatalf("rewritten join created a room instead of joining: %s", joinRec.Body.String())
	}
}

func TestVercelConfigRewritesRoomSubroutesToFunction(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "vercel.json"))
	if err != nil {
		t.Fatalf("read vercel config: %v", err)
	}

	var config struct {
		Rewrites []struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		} `json:"rewrites"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("decode vercel config: %v", err)
	}

	for _, rewrite := range config.Rewrites {
		if rewrite.Source == "/api/rooms/:path*" &&
			(rewrite.Destination == "/api/rooms" || rewrite.Destination == "/api/rooms?path=:path*") {
			return
		}
	}
	t.Fatalf("missing /api/rooms/:path* rewrite in vercel.json: %s", string(data))
}

func TestNilStoreRoutesReturnFailureEnvelopeWithoutPanic(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Restaurants: FakeRestaurantProvider{}})

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{name: "create", method: http.MethodPost, path: "/api/rooms", status: http.StatusInternalServerError},
		{name: "snapshot", method: http.MethodGet, path: "/api/rooms/ABC123", status: http.StatusInternalServerError},
		{name: "join", method: http.MethodPost, path: "/api/rooms/ABC123/join", status: http.StatusInternalServerError},
		{name: "search", method: http.MethodPost, path: "/api/rooms/ABC123/search", body: `{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`, status: http.StatusInternalServerError},
		{name: "recommendations", method: http.MethodPost, path: "/api/rooms/ABC123/recommendations", status: http.StatusInternalServerError},
		{name: "type vote", method: http.MethodPost, path: "/api/rooms/ABC123/votes/type", body: `{"participantId":"p1","typeId":"type-hotpot","vote":"avoid"}`, status: http.StatusInternalServerError},
		{name: "restaurant override", method: http.MethodPost, path: "/api/rooms/ABC123/votes/restaurant", body: `{"participantId":"p1","restaurantId":"amap:test-hotpot","override":"remove"}`, status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serveWithoutPanic(t, server, tt.method, tt.path, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
			}
			assertErrorCode(t, rec.Body.Bytes(), domain.ErrorProvider)
		})
	}
}

func TestNilRestaurantProviderSearchReturnsFailureEnvelopeWithoutPanic(t *testing.T) {
	store := roomstore.NewMemoryStore()
	room, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", time.Now())
	if err := store.Save(context.Background(), room, domain.RoomTTL); err != nil {
		t.Fatalf("save room: %v", err)
	}
	server := NewServer(Config{AppURL: "https://app.test", Store: store})

	rec := serveWithoutPanic(t, server, http.MethodPost, "/api/rooms/ABC123/search", `{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), domain.ErrorProvider)
}

func TestCreateRetriesRoomIDCollision(t *testing.T) {
	store := roomstore.NewMemoryStore()
	existing, _ := domain.NewRoom("ABC123", "https://app.test/room/ABC123", time.Now())
	if err := store.Save(context.Background(), existing, domain.RoomTTL); err != nil {
		t.Fatalf("save existing room: %v", err)
	}
	server := NewServer(Config{
		AppURL:      "https://app.test",
		Store:       store,
		Restaurants: FakeRestaurantProvider{},
	})
	ids := []string{"ABC123", "DEF456"}
	server.newRoomID = func() (string, error) {
		next := ids[0]
		ids = ids[1:]
		return next, nil
	}

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if body.Data["roomId"] != "DEF456" {
		t.Fatalf("roomId = %v, want DEF456", body.Data["roomId"])
	}
}

func TestRoomsPrefixRequiresPathBoundary(t *testing.T) {
	store := roomstore.NewMemoryStore()
	room, _ := domain.NewRoom("XYZ", "https://app.test/room/XYZ", time.Now())
	if err := store.Save(context.Background(), room, domain.RoomTTL); err != nil {
		t.Fatalf("save room: %v", err)
	}
	server := NewServer(Config{
		AppURL:      "https://app.test",
		Store:       store,
		Restaurants: FakeRestaurantProvider{},
	})

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/roomsXYZ", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), domain.ErrorValidation)
}

type envelope struct {
	OK    bool           `json:"ok"`
	Data  map[string]any `json:"data"`
	Error any            `json:"error"`
}

func decodeRoomResponse(t *testing.T, data []byte) domain.Room {
	t.Helper()

	var body struct {
		OK   bool `json:"ok"`
		Data struct {
			Room domain.Room `json:"room"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode room envelope: %v", err)
	}
	if !body.OK {
		t.Fatalf("ok = false, want true: %s", string(data))
	}
	return body.Data.Room
}

type recordingRestaurantProvider struct {
	calls int
}

func (p *recordingRestaurantProvider) SearchAround(ctx context.Context, lat float64, lng float64, radiusKM int, limit int) ([]domain.Restaurant, error) {
	p.calls++
	return FakeRestaurantProvider{}.SearchAround(ctx, lat, lng, radiusKM, limit)
}

func serveWithoutPanic(t *testing.T, server *Server, method string, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("ServeHTTP panicked: %v", recovered)
		}
	}()

	rec := httptest.NewRecorder()
	var requestBody *bytes.Buffer
	if body == "" {
		requestBody = bytes.NewBuffer(nil)
	} else {
		requestBody = bytes.NewBufferString(body)
	}
	server.ServeHTTP(rec, httptest.NewRequest(method, path, requestBody))
	return rec
}

func assertErrorCode(t *testing.T, data []byte, code string) {
	t.Helper()

	var body struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if body.OK {
		t.Fatalf("ok = true, want false: %s", string(data))
	}
	if body.Error.Code != code {
		t.Fatalf("error code = %q, want %q; body = %s", body.Error.Code, code, string(data))
	}
}
