package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/llm"
)

func useMockProviders() bool {
	return os.Getenv("USE_MOCK_PROVIDERS") == "true"
}

func safeLLMBaseURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return false
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return false
	}
	if !validURLHostPort(parsed) {
		return false
	}

	host := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return false
	}
	if strings.Contains(host, "%") {
		return false
	}
	if alternateNumericIPv4Host(host) {
		return false
	}

	if addr, err := netip.ParseAddr(host); err == nil {
		return safeResolvedAddr(addr)
	}
	return true
}

func validURLHostPort(parsed *url.URL) bool {
	port := parsed.Port()
	if port != "" {
		_, err := strconv.ParseUint(port, 10, 16)
		return err == nil
	}

	rawHost := parsed.Host
	if strings.HasPrefix(rawHost, "[") {
		closing := strings.LastIndex(rawHost, "]")
		return closing >= 0 && rawHost[closing+1:] == ""
	}
	return !strings.Contains(rawHost, ":")
}

func alternateNumericIPv4Host(host string) bool {
	parts := strings.Split(strings.ToLower(host), ".")
	if len(parts) == 0 || len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		if part == "" || !numericIPv4Part(part) {
			return false
		}
	}
	_, err := netip.ParseAddr(host)
	return err != nil
}

func numericIPv4Part(part string) bool {
	if strings.HasPrefix(part, "0x") {
		if len(part) == 2 {
			return false
		}
		_, err := strconv.ParseUint(part[2:], 16, 32)
		return err == nil
	}
	_, err := strconv.ParseUint(part, 10, 32)
	return err == nil
}

var llmSpecialUsePrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("255.255.255.255/32"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2001:2::/48"),
	netip.MustParsePrefix("2001:10::/28"),
	netip.MustParsePrefix("2001:20::/28"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
}

func safeResolvedAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	return !specialUseAddr(addr) &&
		!addr.IsLoopback() &&
		!addr.IsPrivate() &&
		!addr.IsUnspecified() &&
		!addr.IsLinkLocalUnicast() &&
		!addr.IsMulticast()
}

func specialUseAddr(addr netip.Addr) bool {
	for _, prefix := range llmSpecialUsePrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

type LLMTagger struct {
	HTTPClient *http.Client
}

type compactRestaurant struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Categories []string `json:"categories"`
	Tags       []string `json:"tags"`
}

func (t LLMTagger) Enhance(ctx context.Context, restaurants []domain.Restaurant, apiKey string, baseURL string, model string) (llm.EnhancementResult, error) {
	compact, err := compactRestaurants(restaurants)
	if err != nil {
		return llm.EnhancementResult{}, err
	}
	return llm.NewClient(baseURL, apiKey, model, t.httpClient()).EnhanceTags(ctx, compact)
}

func (t LLMTagger) httpClient() *http.Client {
	if t.HTTPClient != nil {
		return t.HTTPClient
	}
	dialer := &net.Dialer{
		Timeout:   20 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			DialContext: safeDialContext(dialer, net.DefaultResolver),
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func safeDialContext(dialer *net.Dialer, resolver *net.Resolver) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		host = strings.TrimSuffix(host, ".")
		if strings.Contains(host, "%") {
			return nil, fmt.Errorf("unsafe LLM host %q", host)
		}

		if addr, err := netip.ParseAddr(host); err == nil {
			if !safeResolvedAddr(addr) {
				return nil, fmt.Errorf("unsafe LLM resolved address %s", addr)
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
		}

		addrs, err := resolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("LLM host %q resolved no addresses", host)
		}
		for _, addr := range addrs {
			if !safeResolvedAddr(addr) {
				return nil, fmt.Errorf("unsafe LLM resolved address %s", addr)
			}
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].String(), port))
	}
}

func compactRestaurants(restaurants []domain.Restaurant) (string, error) {
	payload := struct {
		Restaurants []compactRestaurant `json:"restaurants"`
	}{
		Restaurants: make([]compactRestaurant, 0, len(restaurants)),
	}

	for _, restaurant := range restaurants {
		payload.Restaurants = append(payload.Restaurants, compactRestaurant{
			ID:         restaurant.ID,
			Name:       restaurant.Name,
			Categories: restaurant.Categories,
			Tags:       restaurant.Tags,
		})
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
