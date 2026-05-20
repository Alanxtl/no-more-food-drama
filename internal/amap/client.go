package amap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

type SearchRequest struct {
	Lat          float64
	Lng          float64
	RadiusMeters int
	Limit        int
}

type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewClient(apiKey, baseURL string, client *http.Client) *Client {
	if client == nil {
		client = http.DefaultClient
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (c *Client) SearchAround(ctx context.Context, request SearchRequest) ([]domain.Restaurant, error) {
	values := url.Values{}
	values.Set("key", c.apiKey)
	values.Set("location", fmt.Sprintf("%f,%f", request.Lng, request.Lat))
	values.Set("radius", strconv.Itoa(request.RadiusMeters))
	values.Set("types", "050000")
	values.Set("page_size", strconv.Itoa(normalizePageSize(request.Limit)))
	values.Set("show_fields", "business")

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v5/place/around?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	response, err := c.client.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode > http.StatusMultipleChoices-1 {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("amap request failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded amapAroundResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if decoded.Status != "1" {
		return nil, fmt.Errorf("amap search failed: %s", decoded.Info)
	}

	return mapPOIs(decoded.POIs), nil
}

type amapAroundResponse struct {
	Status string    `json:"status"`
	Info   string    `json:"info"`
	POIs   []amapPOI `json:"pois"`
}

type amapPOI struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Address  address `json:"address"`
	Location string  `json:"location"`
	Distance string  `json:"distance"`
	Type     string  `json:"type"`
	BizExt   struct {
		Rating string `json:"rating"`
		Cost   string `json:"cost"`
	} `json:"biz_ext"`
}

func mapPOIs(pois []amapPOI) []domain.Restaurant {
	restaurants := make([]domain.Restaurant, 0, len(pois))
	for _, poi := range pois {
		lng, lat, ok := parseLocation(poi.Location)
		if !ok {
			continue
		}
		distance, _ := strconv.Atoi(poi.Distance)
		rating, _ := strconv.ParseFloat(poi.BizExt.Rating, 64)
		avgPriceCNY, _ := strconv.Atoi(poi.BizExt.Cost)

		restaurants = append(restaurants, domain.Restaurant{
			ID:             "amap:" + poi.ID,
			Provider:       "amap",
			ProviderID:     poi.ID,
			Name:           poi.Name,
			Address:        poi.Address.String(),
			Lat:            lat,
			Lng:            lng,
			DistanceMeters: distance,
			Rating:         rating,
			AvgPriceCNY:    avgPriceCNY,
			Categories:     splitCategories(poi.Type),
			TypeIDs:        []string{},
			Tags:           []string{},
		})
	}
	return restaurants
}

type address []string

func (a *address) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		*a = normalizeAddressParts([]string{value})
		return nil
	}

	var values []string
	if err := json.Unmarshal(data, &values); err == nil {
		*a = normalizeAddressParts(values)
		return nil
	}

	if string(data) == "null" {
		*a = nil
		return nil
	}
	return fmt.Errorf("unsupported address shape: %s", string(data))
}

func (a address) String() string {
	return strings.Join(a, " ")
}

func normalizeAddressParts(parts []string) []string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			normalized = append(normalized, part)
		}
	}
	return normalized
}

func parseLocation(location string) (float64, float64, bool) {
	parts := strings.Split(location, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	lng, lngErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lat, latErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if lngErr != nil || latErr != nil {
		return 0, 0, false
	}
	return lng, lat, true
}

func splitCategories(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ";")
	categories := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			categories = append(categories, part)
		}
	}
	return categories
}

func normalizePageSize(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 25 {
		return 25
	}
	return limit
}
