package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/amap"
	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type recordingAmapClient struct {
	request amap.SearchRequest
}

func (c *recordingAmapClient) SearchAround(ctx context.Context, request amap.SearchRequest) ([]domain.Restaurant, error) {
	c.request = request
	return []domain.Restaurant{{ID: "amap:test", Name: "测试餐厅"}}, nil
}

func TestAmapRestaurantProviderConvertsRadiusToMeters(t *testing.T) {
	client := &recordingAmapClient{}
	provider := AmapRestaurantProvider{Client: client}

	restaurants, err := provider.SearchAround(context.Background(), 23.09, 113.32, 3, 20)
	if err != nil {
		t.Fatalf("SearchAround returned error: %v", err)
	}

	if len(restaurants) != 1 {
		t.Fatalf("restaurants length = %d", len(restaurants))
	}
	if client.request.RadiusMeters != 3000 || client.request.Limit != 20 {
		t.Fatalf("request = %#v", client.request)
	}
}

func TestAmapRestaurantProviderReturnsErrorWithoutClient(t *testing.T) {
	provider := AmapRestaurantProvider{}

	_, err := provider.SearchAround(context.Background(), 23.09, 113.32, 3, 20)
	if err == nil {
		t.Fatal("SearchAround returned nil error, want provider configuration error")
	}
}

func TestLLMTaggerDefaultHTTPClientHasBoundedTimeout(t *testing.T) {
	client := LLMTagger{}.httpClient()

	if client.Timeout <= 0 {
		t.Fatalf("timeout = %s, want bounded default", client.Timeout)
	}
	if client.Timeout > 20*time.Second {
		t.Fatalf("timeout = %s, want at most 20s", client.Timeout)
	}
}

func TestLLMTaggerDefaultHTTPClientDisablesRedirects(t *testing.T) {
	client := LLMTagger{}.httpClient()
	if client.CheckRedirect == nil {
		t.Fatalf("CheckRedirect is nil, want redirects disabled")
	}

	request, err := http.NewRequest(http.MethodGet, "https://127.0.0.1/v1/chat/completions", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	err = client.CheckRedirect(request, []*http.Request{})
	if !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect error = %v, want %v", err, http.ErrUseLastResponse)
	}
}

func TestLLMTaggerDefaultHTTPClientDisablesProxy(t *testing.T) {
	client := LLMTagger{}.httpClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatalf("Proxy is configured, want nil")
	}
}

func TestSafeLLMBaseURLRejectsQueryAndFragment(t *testing.T) {
	tests := []string{
		"https://api.example.com/v1?x=y",
		"https://api.example.com/v1#frag",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if safeLLMBaseURL(raw) {
				t.Fatalf("safeLLMBaseURL(%q) = true, want false", raw)
			}
		})
	}
}

func TestSafeLLMBaseURLRejectsAlternateNumericIPv4Forms(t *testing.T) {
	tests := []string{
		"https://2130706433/v1",
		"https://0177.0.0.1/v1",
		"https://0x7f.0.0.1/v1",
		"https://127.1/v1",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if safeLLMBaseURL(raw) {
				t.Fatalf("safeLLMBaseURL(%q) = true, want false", raw)
			}
		})
	}
}

func TestSafeLLMBaseURLAllowsPublicHTTPSHostname(t *testing.T) {
	if !safeLLMBaseURL("https://api.example.com/v1") {
		t.Fatalf("safeLLMBaseURL rejected public HTTPS hostname")
	}
}

func TestSafeResolvedAddrRejectsUnsafeIPs(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{name: "loopback", addr: "127.0.0.1"},
		{name: "private", addr: "10.0.0.1"},
		{name: "unspecified", addr: "0.0.0.0"},
		{name: "link local", addr: "169.254.1.1"},
		{name: "multicast", addr: "224.0.0.1"},
		{name: "mapped loopback", addr: "::ffff:127.0.0.1"},
		{name: "mapped private", addr: "::ffff:10.0.0.1"},
		{name: "mapped link local", addr: "::ffff:169.254.1.1"},
		{name: "carrier grade nat", addr: "100.64.0.1"},
		{name: "ietf protocol assignments", addr: "192.0.0.1"},
		{name: "six to four relay anycast", addr: "192.88.99.1"},
		{name: "benchmarking", addr: "198.18.0.1"},
		{name: "documentation ipv4", addr: "192.0.2.1"},
		{name: "documentation test net two", addr: "198.51.100.1"},
		{name: "documentation test net three", addr: "203.0.113.1"},
		{name: "reserved future use", addr: "240.0.0.1"},
		{name: "broadcast", addr: "255.255.255.255"},
		{name: "nat64 well known prefix", addr: "64:ff9b::a00:1"},
		{name: "nat64 local use", addr: "64:ff9b:1::1"},
		{name: "discard only ipv6", addr: "100::1"},
		{name: "teredo", addr: "2001::1"},
		{name: "ietf protocol assignments ipv6 one", addr: "2001:1::1"},
		{name: "benchmarking ipv6", addr: "2001:2::1"},
		{name: "ietf protocol assignments ipv6 three", addr: "2001:3::1"},
		{name: "orchid", addr: "2001:10::1"},
		{name: "orchid v2", addr: "2001:20::1"},
		{name: "six to four ipv6", addr: "2002::1"},
		{name: "documentation ipv6", addr: "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := netip.MustParseAddr(tt.addr)
			if safeResolvedAddr(addr) {
				t.Fatalf("safeResolvedAddr(%s) = true, want false", tt.addr)
			}
		})
	}
}

func TestSafeResolvedAddrAllowsPublicIP(t *testing.T) {
	tests := []string{
		"93.184.216.34",
		"2606:4700:4700::1111",
	}

	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			addr := netip.MustParseAddr(raw)
			if !safeResolvedAddr(addr) {
				t.Fatalf("safeResolvedAddr(%s) = false, want true", addr)
			}
		})
	}
}
