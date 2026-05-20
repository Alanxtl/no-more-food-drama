package handler

import (
	"net/http"
	"os"
	"sync"

	"github.com/Alanxtl/no-more-food-drama/internal/httpapi"
	"github.com/Alanxtl/no-more-food-drama/internal/roomstore"
)

var (
	serverOnce sync.Once
	server     *httpapi.Server
)

func Handler(w http.ResponseWriter, r *http.Request) {
	serverOnce.Do(func() {
		server = httpapi.NewServer(httpapi.Config{
			AppURL:      env("NEXT_PUBLIC_APP_URL", "http://localhost:3000"),
			Store:       roomstore.NewMemoryStore(),
			Restaurants: httpapi.FakeRestaurantProvider{},
		})
	})
	server.ServeHTTP(w, r)
}

func env(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
