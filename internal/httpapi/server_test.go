package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/llm"
	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
)

func TestCreateAndJoinRoom(t *testing.T) {
	server := NewServer(Config{
		AppURL:      "https://app.test",
		Store:       roomstore.NewMemoryStore(),
		Restaurants: FakeRestaurantProvider{},
		Tagger:      FakeTagger{},
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
		Tagger:      FakeTagger{},
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

func TestTagEndpointMergesLLMEnhancements(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: FakeTagger{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)

	searchBody := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	searchReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", searchBody)
	server.ServeHTTP(httptest.NewRecorder(), searchReq)

	tagBody := bytes.NewBufferString(`{"apiKey":"sk-test","baseUrl":"https://api.example.com/v1","model":"deepseek-chat"}`)
	tagReq := httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/tag", tagBody)
	tagRec := httptest.NewRecorder()
	server.ServeHTTP(tagRec, tagReq)

	if tagRec.Code != http.StatusOK {
		t.Fatalf("tag status = %d body = %s", tagRec.Code, tagRec.Body.String())
	}
	if !bytes.Contains(tagRec.Body.Bytes(), []byte("漂亮饭")) {
		t.Fatalf("tag body missing LLM tag: %s", tagRec.Body.String())
	}
}

func TestTagEndpointRejectsUnsafeBaseURLsWithoutCallingTagger(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{name: "malformed", baseURL: "://broken"},
		{name: "http", baseURL: "http://api.example.com/v1"},
		{name: "userinfo", baseURL: "https://user:pass@api.example.com/v1"},
		{name: "invalid host port", baseURL: "https://api.example.com:bad/v1"},
		{name: "invalid loopback port", baseURL: "https://127.0.0.1:bad/v1"},
		{name: "force query", baseURL: "https://api.example.com/v1?"},
		{name: "localhost", baseURL: "https://localhost/v1"},
		{name: "localhost suffix", baseURL: "https://foo.localhost/v1"},
		{name: "loopback ipv4", baseURL: "https://127.0.0.1/v1"},
		{name: "private ipv4", baseURL: "https://10.0.0.1/v1"},
		{name: "unspecified ipv4", baseURL: "https://0.0.0.0/v1"},
		{name: "link local ipv4", baseURL: "https://169.254.1.1/v1"},
		{name: "multicast ipv4", baseURL: "https://224.0.0.1/v1"},
		{name: "loopback ipv6", baseURL: "https://[::1]/v1"},
		{name: "mapped ipv6 loopback", baseURL: "https://[::ffff:127.0.0.1]/v1"},
		{name: "mapped ipv6 private", baseURL: "https://[::ffff:10.0.0.1]/v1"},
		{name: "mapped ipv6 link local", baseURL: "https://[::ffff:169.254.1.1]/v1"},
		{name: "scoped ipv6 link local", baseURL: "https://[fe80::1%25eth0]/v1"},
		{name: "scoped ipv6 global", baseURL: "https://[2001:db8::1%25eth0]/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tagger := &recordingTagger{}
			server, roomID := newTaggedSearchRoom(t, tagger)

			body := bytes.NewBufferString(`{"apiKey":"sk-test","baseUrl":"` + tt.baseURL + `","model":"deepseek-chat"}`)
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/tag", body))

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
			}
			if tagger.calls != 0 {
				t.Fatalf("tagger calls = %d, want 0", tagger.calls)
			}
			assertErrorCode(t, rec.Body.Bytes(), domain.ErrorValidation)
		})
	}
}

func TestTagEndpointAcceptsSafeHTTPSBaseURL(t *testing.T) {
	tagger := &recordingTagger{}
	server, roomID := newTaggedSearchRoom(t, tagger)

	body := bytes.NewBufferString(`{"apiKey":"sk-test","baseUrl":"https://api.example.com/v1","model":"deepseek-chat"}`)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/tag", body))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if tagger.calls != 1 {
		t.Fatalf("tagger calls = %d, want 1", tagger.calls)
	}
}

func TestTagEndpointRejectsEmptyRestaurantRoomWithoutCallingTagger(t *testing.T) {
	tagger := &recordingTagger{}
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: tagger})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)

	tagBody := bytes.NewBufferString(`{"apiKey":"sk-test","baseUrl":"https://api.example.com/v1","model":"deepseek-chat"}`)
	tagRec := httptest.NewRecorder()
	server.ServeHTTP(tagRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/tag", tagBody))

	if tagRec.Code != http.StatusBadRequest {
		t.Fatalf("tag status = %d body = %s", tagRec.Code, tagRec.Body.String())
	}
	if tagger.calls != 0 {
		t.Fatalf("tagger calls = %d, want 0", tagger.calls)
	}
	assertErrorCode(t, tagRec.Body.Bytes(), domain.ErrorValidation)
}

func TestTagEndpointUsesRequestLocalConfigAndRefreshesRoomState(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	later := now.Add(10 * time.Minute)
	store := roomstore.NewMemoryStore()
	store.SetNow(now)
	tagger := &recordingTagger{}
	server := NewServer(Config{AppURL: "https://app.test", Store: store, Restaurants: FakeRestaurantProvider{}, Tagger: tagger})
	server.now = func() time.Time { return now }

	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)
	creatorID := createBody.Data["participantId"].(string)

	searchBody := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	searchRec := httptest.NewRecorder()
	server.ServeHTTP(searchRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", searchBody))
	searchRoom := decodeRoomResponse(t, searchRec.Body.Bytes())
	joinRec := httptest.NewRecorder()
	server.ServeHTTP(joinRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/join", nil))
	var joinBody struct {
		Data struct {
			ParticipantID string `json:"participantId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(joinRec.Body.Bytes(), &joinBody); err != nil {
		t.Fatalf("decode join: %v", err)
	}
	voteAllTypes(t, server, roomID, creatorID, searchRoom.Types)
	voteAllTypes(t, server, roomID, joinBody.Data.ParticipantID, searchRoom.Types)
	server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/recommendations", nil))

	before, err := store.Get(context.Background(), roomID)
	if err != nil {
		t.Fatalf("get before tag: %v", err)
	}
	if len(before.Recommendations) == 0 {
		t.Fatalf("expected recommendations before tag")
	}

	store.SetNow(later)
	server.now = func() time.Time { return later }
	tagBody := bytes.NewBufferString(`{"apiKey":"  sk-request  ","baseUrl":"  https://api.example.com/v1  ","model":"  deepseek-chat  "}`)
	tagRec := httptest.NewRecorder()
	server.ServeHTTP(tagRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/tag", tagBody))

	if tagRec.Code != http.StatusOK {
		t.Fatalf("tag status = %d body = %s", tagRec.Code, tagRec.Body.String())
	}
	if tagger.calls != 1 {
		t.Fatalf("tagger calls = %d, want 1", tagger.calls)
	}
	if tagger.apiKey != "sk-request" || tagger.baseURL != "https://api.example.com/v1" || tagger.model != "deepseek-chat" {
		t.Fatalf("tagger config = apiKey %q baseURL %q model %q", tagger.apiKey, tagger.baseURL, tagger.model)
	}

	room := decodeRoomResponse(t, tagRec.Body.Bytes())
	if room.Status != domain.StatusFiltering {
		t.Fatalf("status = %q, want %q", room.Status, domain.StatusFiltering)
	}
	if len(room.Recommendations) != 0 {
		t.Fatalf("recommendations length = %d, want 0", len(room.Recommendations))
	}
	if room.Version != before.Version+1 {
		t.Fatalf("version = %d, want %d", room.Version, before.Version+1)
	}
	if !room.ExpiresAt.After(before.ExpiresAt) {
		t.Fatalf("ExpiresAt = %s, want after %s", room.ExpiresAt, before.ExpiresAt)
	}

	responseBody := tagRec.Body.String()
	if strings.Contains(responseBody, "sk-request") ||
		strings.Contains(responseBody, "https://api.example.com/v1") ||
		strings.Contains(responseBody, "deepseek-chat") {
		t.Fatalf("response leaked request-local LLM config: %s", responseBody)
	}
	stored, err := store.Get(context.Background(), roomID)
	if err != nil {
		t.Fatalf("get stored room: %v", err)
	}
	storedJSON, err := json.Marshal(stored)
	if err != nil {
		t.Fatalf("marshal stored room: %v", err)
	}
	if bytes.Contains(storedJSON, []byte("sk-request")) ||
		bytes.Contains(storedJSON, []byte("https://api.example.com/v1")) ||
		bytes.Contains(storedJSON, []byte("deepseek-chat")) {
		t.Fatalf("stored room leaked request-local LLM config: %s", string(storedJSON))
	}
}

func TestTagEndpointRejectsStaleLLMResultAfterConcurrentSearch(t *testing.T) {
	store := roomstore.NewMemoryStore()
	tagger := &concurrentSearchTagger{}
	server := NewServer(Config{AppURL: "https://app.test", Store: store, Restaurants: FakeRestaurantProvider{}, Tagger: tagger})

	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)

	searchBody := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	searchRec := httptest.NewRecorder()
	server.ServeHTTP(searchRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", searchBody))
	if searchRec.Code != http.StatusOK {
		t.Fatalf("search status = %d body = %s", searchRec.Code, searchRec.Body.String())
	}
	beforeTag, err := store.Get(context.Background(), roomID)
	if err != nil {
		t.Fatalf("get before tag: %v", err)
	}

	tagger.onEnhance = func() {
		concurrentBody := bytes.NewBufferString(`{"lat":24.01,"lng":114.02,"radiusKm":5,"limit":20}`)
		concurrentRec := httptest.NewRecorder()
		server.ServeHTTP(concurrentRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", concurrentBody))
		if concurrentRec.Code != http.StatusOK {
			t.Fatalf("concurrent search status = %d body = %s", concurrentRec.Code, concurrentRec.Body.String())
		}
	}

	tagBody := bytes.NewBufferString(`{"apiKey":"sk-test","baseUrl":"https://api.example.com/v1","model":"deepseek-chat"}`)
	tagRec := httptest.NewRecorder()
	server.ServeHTTP(tagRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/tag", tagBody))

	if tagRec.Code != http.StatusConflict {
		t.Fatalf("tag status = %d body = %s", tagRec.Code, tagRec.Body.String())
	}
	assertErrorCode(t, tagRec.Body.Bytes(), domain.ErrorValidation)

	afterTag, err := store.Get(context.Background(), roomID)
	if err != nil {
		t.Fatalf("get after tag: %v", err)
	}
	if afterTag.Version != beforeTag.Version+1 {
		t.Fatalf("version = %d, want concurrent search version %d", afterTag.Version, beforeTag.Version+1)
	}
	if afterTag.SearchConfig == nil || afterTag.SearchConfig.Lat != 24.01 || afterTag.SearchConfig.Lng != 114.02 {
		t.Fatalf("search config = %#v, want concurrent search config", afterTag.SearchConfig)
	}
	if restaurantHasTag(afterTag.Restaurants, "amap:test-sushi", "漂亮饭") {
		t.Fatalf("stale LLM tag was applied after conflict: %#v", afterTag.Restaurants)
	}
}

func TestSearchMissingRoomReturnsNotFoundWithoutCallingProvider(t *testing.T) {
	provider := &recordingRestaurantProvider{}
	server := NewServer(Config{
		AppURL:      "https://app.test",
		Store:       roomstore.NewMemoryStore(),
		Restaurants: provider,
		Tagger:      FakeTagger{},
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
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: FakeTagger{}})
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
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: FakeTagger{}})
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

func TestRecommendationsRequireBothParticipantsToFinishTypeVotes(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: FakeTagger{}})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	_ = json.Unmarshal(createRec.Body.Bytes(), &createBody)
	roomID := createBody.Data["roomId"].(string)
	creatorID := createBody.Data["participantId"].(string)

	joinRec := httptest.NewRecorder()
	server.ServeHTTP(joinRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/join", nil))
	var joinBody struct {
		Data struct {
			ParticipantID string `json:"participantId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(joinRec.Body.Bytes(), &joinBody); err != nil {
		t.Fatalf("decode join: %v", err)
	}
	partnerID := joinBody.Data.ParticipantID

	searchBody := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	searchRec := httptest.NewRecorder()
	server.ServeHTTP(searchRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", searchBody))
	if searchRec.Code != http.StatusOK {
		t.Fatalf("search status = %d body = %s", searchRec.Code, searchRec.Body.String())
	}
	room := decodeRoomResponse(t, searchRec.Body.Bytes())
	if len(room.Types) == 0 {
		t.Fatal("search returned no types")
	}

	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/recommendations", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("recommendations status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec.Body.Bytes(), domain.ErrorValidation)

	voteAllTypes(t, server, roomID, creatorID, room.Types)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/recommendations", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("recommendations after one participant status = %d body = %s", rec.Code, rec.Body.String())
	}

	voteAllTypes(t, server, roomID, partnerID, room.Types)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/recommendations", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("recommendations after both participants status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := decodeRoomResponse(t, rec.Body.Bytes()).Status; got != domain.StatusResults {
		t.Fatalf("status = %q, want %q", got, domain.StatusResults)
	}
}

func TestVoteEndpointsValidatePayloads(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: FakeTagger{}})
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
	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: FakeTagger{}})
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
		Tagger:      FakeTagger{},
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
		if rewrite.Source == "^/api/rooms/(.*)$" && rewrite.Destination == "/api/rooms?path=$1" {
			return
		}
	}
	t.Fatalf("missing room subroute rewrite in vercel.json: %s", string(data))
}

func TestNilStoreRoutesReturnFailureEnvelopeWithoutPanic(t *testing.T) {
	server := NewServer(Config{AppURL: "https://app.test", Restaurants: FakeRestaurantProvider{}, Tagger: FakeTagger{}})

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
		{name: "tag", method: http.MethodPost, path: "/api/rooms/ABC123/tag", body: `{"apiKey":"sk-test","baseUrl":"https://api.example.com/v1","model":"deepseek-chat"}`, status: http.StatusInternalServerError},
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
	server := NewServer(Config{AppURL: "https://app.test", Store: store, Tagger: FakeTagger{}})

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
		Tagger:      FakeTagger{},
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
		Tagger:      FakeTagger{},
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

type recordingTagger struct {
	calls   int
	apiKey  string
	baseURL string
	model   string
}

func (t *recordingTagger) Enhance(ctx context.Context, restaurants []domain.Restaurant, apiKey string, baseURL string, model string) (llm.EnhancementResult, error) {
	t.calls++
	t.apiKey = apiKey
	t.baseURL = baseURL
	t.model = model
	return FakeTagger{}.Enhance(ctx, restaurants, apiKey, baseURL, model)
}

type concurrentSearchTagger struct {
	onEnhance func()
}

func (t *concurrentSearchTagger) Enhance(ctx context.Context, restaurants []domain.Restaurant, apiKey string, baseURL string, model string) (llm.EnhancementResult, error) {
	if t.onEnhance != nil {
		t.onEnhance()
	}
	return FakeTagger{}.Enhance(ctx, restaurants, apiKey, baseURL, model)
}

func restaurantHasTag(restaurants []domain.Restaurant, restaurantID string, tag string) bool {
	for _, restaurant := range restaurants {
		if restaurant.ID != restaurantID {
			continue
		}
		for _, got := range restaurant.Tags {
			if got == tag {
				return true
			}
		}
	}
	return false
}

func newTaggedSearchRoom(t *testing.T, tagger Tagger) (*Server, string) {
	t.Helper()

	server := NewServer(Config{AppURL: "https://app.test", Store: roomstore.NewMemoryStore(), Restaurants: FakeRestaurantProvider{}, Tagger: tagger})
	createRec := httptest.NewRecorder()
	server.ServeHTTP(createRec, httptest.NewRequest(http.MethodPost, "/api/rooms", nil))
	var createBody envelope
	if err := json.Unmarshal(createRec.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	roomID := createBody.Data["roomId"].(string)

	searchBody := bytes.NewBufferString(`{"lat":23.09,"lng":113.32,"radiusKm":3,"limit":20}`)
	searchRec := httptest.NewRecorder()
	server.ServeHTTP(searchRec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/search", searchBody))
	if searchRec.Code != http.StatusOK {
		t.Fatalf("search status = %d body = %s", searchRec.Code, searchRec.Body.String())
	}
	return server, roomID
}

func voteAllTypes(t *testing.T, server *Server, roomID string, participantID string, types []domain.FoodType) {
	t.Helper()

	for _, foodType := range types {
		body := bytes.NewBufferString(`{"participantId":"` + participantID + `","typeId":"` + foodType.ID + `","vote":"neutral"}`)
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/rooms/"+roomID+"/votes/type", body))
		if rec.Code != http.StatusOK {
			t.Fatalf("vote %s status = %d body = %s", foodType.ID, rec.Code, rec.Body.String())
		}
	}
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
