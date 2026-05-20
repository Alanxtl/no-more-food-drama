package httpapi

import (
	"context"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/llm"
)

type FakeRestaurantProvider struct{}

func (FakeRestaurantProvider) SearchAround(ctx context.Context, lat float64, lng float64, radiusKM int, limit int) ([]domain.Restaurant, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	open := true
	restaurants := []domain.Restaurant{
		{
			ID:             "amap:test-sushi",
			Provider:       "amap",
			ProviderID:     "test-sushi",
			Name:           "鮨小野",
			Address:        "测试路 1 号",
			Lat:            lat,
			Lng:            lng,
			DistanceMeters: 650,
			Rating:         4.7,
			AvgPriceCNY:    128,
			OpenNow:        &open,
			Categories:     []string{"餐饮服务", "日本料理"},
		},
		{
			ID:             "amap:test-hotpot",
			Provider:       "amap",
			ProviderID:     "test-hotpot",
			Name:           "热辣火锅",
			Address:        "测试路 2 号",
			Lat:            lat,
			Lng:            lng,
			DistanceMeters: 900,
			Rating:         4.5,
			AvgPriceCNY:    98,
			OpenNow:        &open,
			Categories:     []string{"餐饮服务", "火锅"},
		},
	}
	if limit > 0 && limit < len(restaurants) {
		return restaurants[:limit], nil
	}
	return restaurants, nil
}

type FakeTagger struct{}

func (FakeTagger) Enhance(ctx context.Context, restaurants []domain.Restaurant, apiKey string, baseURL string, model string) (llm.EnhancementResult, error) {
	return llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "amap:test-sushi", TypeIDs: []string{"type-japanese"}, Tags: []string{"漂亮饭", "约会友好"}},
		},
	}, nil
}
