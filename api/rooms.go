package handler

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/amap"
	"github.com/Alanxtl/no-more-food-drama/internal/httpapi"
	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
)

const providerHTTPTimeout = 20 * time.Second

var (
	serverOnce sync.Once
	server     *httpapi.Server
)

func Handler(w http.ResponseWriter, r *http.Request) {
	serverOnce.Do(func() {
		store, restaurants, tagger := runtimeProviders()
		server = httpapi.NewServer(httpapi.Config{
			AppURL:      env("NEXT_PUBLIC_APP_URL", "http://localhost:3000"),
			Store:       store,
			Restaurants: restaurants,
			Tagger:      tagger,
		})
	})
	server.ServeHTTP(w, r)
}

func runtimeProviders() (roomstore.Store, httpapi.RestaurantProvider, httpapi.Tagger) {
	if os.Getenv("USE_MOCK_PROVIDERS") == "true" {
		return roomstore.NewMemoryStore(), httpapi.FakeRestaurantProvider{}, httpapi.FakeTagger{}
	}

	client := providerHTTPClient()
	amapClient := amap.NewClient(requiredEnv("AMAP_API_KEY"), "https://restapi.amap.com", client)
	return roomstore.NewUpstashStore(requiredEnv("UPSTASH_REDIS_REST_URL"), requiredEnv("UPSTASH_REDIS_REST_TOKEN"), client),
		httpapi.AmapRestaurantProvider{Client: amapClient},
		httpapi.LLMTagger{}
}

func providerHTTPClient() *http.Client {
	return &http.Client{Timeout: providerHTTPTimeout}
}

func env(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		panic(name + " is required")
	}
	return value
}
