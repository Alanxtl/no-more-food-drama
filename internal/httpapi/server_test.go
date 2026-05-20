package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

type envelope struct {
	OK    bool           `json:"ok"`
	Data  map[string]any `json:"data"`
	Error any            `json:"error"`
}
