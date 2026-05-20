package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSendsOpenAICompatibleRequest(t *testing.T) {
	var gotAuth string
	var gotContentType string
	var gotPayload struct {
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{
			"choices":[
				{"message":{"content":"{\"restaurants\":[{\"id\":\"r1\",\"typeIds\":[\"type-japanese\"],\"tags\":[\"约会友好\"]}]}"}}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL+"/", "user-key", "deepseek-chat", server.Client())
	result, err := client.EnhanceTags(context.Background(), `{"restaurants":[{"id":"r1","name":"鮨小野"}]}`)
	if err != nil {
		t.Fatalf("EnhanceTags returned error: %v", err)
	}
	if gotAuth != "Bearer user-key" {
		t.Fatalf("authorization = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content type = %q", gotContentType)
	}
	if gotPayload.Model != "deepseek-chat" || gotPayload.Temperature != 0.2 {
		t.Fatalf("payload = %#v", gotPayload)
	}
	if len(gotPayload.Messages) != 2 || gotPayload.Messages[0].Role != "system" || gotPayload.Messages[1].Role != "user" {
		t.Fatalf("messages = %#v", gotPayload.Messages)
	}
	if gotPayload.Messages[1].Content != `{"restaurants":[{"id":"r1","name":"鮨小野"}]}` {
		t.Fatalf("user content = %q", gotPayload.Messages[1].Content)
	}
	if len(result.Restaurants) != 1 || result.Restaurants[0].ID != "r1" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Restaurants[0].TypeIDs) != 1 || result.Restaurants[0].TypeIDs[0] != "type-japanese" {
		t.Fatalf("type IDs = %#v", result.Restaurants[0].TypeIDs)
	}
}

func TestEnhanceTagsReturnsErrorForNoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "user-key", "deepseek-chat", server.Client())
	if _, err := client.EnhanceTags(context.Background(), `{}`); err == nil {
		t.Fatal("EnhanceTags returned nil error")
	}
}

func TestEnhanceTagsReturnsErrorForHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"quota"}`, http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(server.URL, "user-key", "deepseek-chat", server.Client())
	if _, err := client.EnhanceTags(context.Background(), `{}`); err == nil {
		t.Fatal("EnhanceTags returned nil error")
	}
}
