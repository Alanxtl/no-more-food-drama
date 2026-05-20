package handler

import (
	"net/http"
	"testing"
)

func TestProviderHTTPClientHasBoundedTimeout(t *testing.T) {
	client := providerHTTPClient()

	if client.Timeout <= 0 {
		t.Fatalf("timeout = %s, want bounded default", client.Timeout)
	}
	if client.Timeout > providerHTTPTimeout {
		t.Fatalf("timeout = %s, want at most %s", client.Timeout, providerHTTPTimeout)
	}
}

func TestProviderHTTPClientDoesNotUseDefaultClient(t *testing.T) {
	if providerHTTPClient() == http.DefaultClient {
		t.Fatal("providerHTTPClient returned http.DefaultClient")
	}
}
