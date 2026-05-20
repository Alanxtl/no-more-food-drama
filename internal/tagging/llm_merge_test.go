package tagging

import (
	"slices"
	"testing"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/llm"
)

func TestMergeLLMEnhancementsUpdatesOnlyTagsAndTypes(t *testing.T) {
	open := true
	restaurants := []domain.Restaurant{
		{
			ID: "r1", Provider: "amap", ProviderID: "p1", Name: "热辣火锅", Address: "朝阳路 1 号",
			Lat: 39.9, Lng: 116.4, DistanceMeters: 650, Rating: 4.6, PriceLevel: "2",
			AvgPriceCNY: 88, OpenNow: &open, Categories: []string{"火锅"},
			TypeIDs: []string{"type-hotpot"}, Tags: []string{"正餐", "离得近"},
		},
	}
	result := llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "r1", TypeIDs: []string{" type-sichuan ", "type-hotpot"}, Tags: []string{" 重口味 ", "正餐"}},
		},
	}

	merged, _ := MergeLLMEnhancements(restaurants, result)

	if len(merged) != 1 {
		t.Fatalf("merged length = %d", len(merged))
	}
	got := merged[0]
	if got.Provider != "amap" || got.ProviderID != "p1" || got.Address != "朝阳路 1 号" ||
		got.Lat != 39.9 || got.Lng != 116.4 || got.DistanceMeters != 650 ||
		got.Rating != 4.6 || got.PriceLevel != "2" || got.AvgPriceCNY != 88 ||
		got.OpenNow != &open || !slices.Equal(got.Categories, []string{"火锅"}) {
		t.Fatalf("non-classification fields changed: %#v", got)
	}
	if !slices.Equal(got.TypeIDs, []string{"type-sichuan", "type-hotpot"}) {
		t.Fatalf("TypeIDs = %#v", got.TypeIDs)
	}
	assertTagsEqual(t, got.Tags, []string{"正餐", "离得近", "重口味"})
}

func TestMergeLLMEnhancementsIgnoresUnknownIDs(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅", TypeIDs: []string{"type-hotpot"}, Tags: []string{"正餐"}},
	}
	result := llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "missing", TypeIDs: []string{"type-dessert"}, Tags: []string{"下午茶"}},
		},
	}

	merged, _ := MergeLLMEnhancements(restaurants, result)

	if !slices.Equal(merged[0].TypeIDs, []string{"type-hotpot"}) {
		t.Fatalf("TypeIDs = %#v", merged[0].TypeIDs)
	}
	if !slices.Equal(merged[0].Tags, []string{"正餐"}) {
		t.Fatalf("Tags = %#v", merged[0].Tags)
	}
}

func TestMergeLLMEnhancementsFallsBackWhenTypeIDsAreBlank(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅", TypeIDs: []string{"type-hotpot"}, Tags: []string{"正餐"}},
	}
	result := llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "r1", TypeIDs: []string{" ", ""}, Tags: []string{"重口味"}},
		},
	}

	merged, _ := MergeLLMEnhancements(restaurants, result)

	if !slices.Equal(merged[0].TypeIDs, []string{"type-hotpot"}) {
		t.Fatalf("TypeIDs = %#v", merged[0].TypeIDs)
	}
	assertTagsEqual(t, merged[0].Tags, []string{"正餐", "重口味"})
}

func TestMergeLLMEnhancementsNormalizesFallbackTypeIDsBeforeAggregation(t *testing.T) {
	restaurants := []domain.Restaurant{
		{
			ID: "r1", Name: "热辣火锅", DistanceMeters: 500, Rating: 4.6, AvgPriceCNY: 80,
			TypeIDs: []string{"type-hotpot", " type-hotpot ", "type-hotpot"}, Tags: []string{"正餐"},
		},
	}
	result := llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "r1", TypeIDs: []string{" ", ""}},
		},
	}

	merged, types := MergeLLMEnhancements(restaurants, result)

	if !slices.Equal(merged[0].TypeIDs, []string{"type-hotpot"}) {
		t.Fatalf("TypeIDs = %#v", merged[0].TypeIDs)
	}
	hotpot := findType(t, types, "type-hotpot")
	if !slices.Equal(hotpot.RestaurantIDs, []string{"r1"}) {
		t.Fatalf("restaurant IDs = %#v", hotpot.RestaurantIDs)
	}
	if hotpot.Stats.Count != 1 {
		t.Fatalf("count = %d", hotpot.Stats.Count)
	}
	if hotpot.Stats.AvgRating != 4.6 {
		t.Fatalf("AvgRating = %v", hotpot.Stats.AvgRating)
	}
	if hotpot.Stats.AvgPriceCNY != 80 {
		t.Fatalf("AvgPriceCNY = %d", hotpot.Stats.AvgPriceCNY)
	}
}

func TestMergeLLMEnhancementsMergesDedupesTagsAndIgnoresBlanks(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅", TypeIDs: []string{"type-hotpot"}, Tags: []string{"正餐", "离得近"}},
	}
	result := llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "r1", TypeIDs: []string{"type-hotpot"}, Tags: []string{" 离得近 ", "", "重口味", "重口味"}},
		},
	}

	merged, _ := MergeLLMEnhancements(restaurants, result)

	assertTagsEqual(t, merged[0].Tags, []string{"正餐", "离得近", "重口味"})
}

func TestMergeLLMEnhancementsDoesNotMutateInputSlicesWithSpareCapacity(t *testing.T) {
	typeIDs := make([]string, 1, 4)
	typeIDs[0] = "type-hotpot"
	tags := make([]string, 1, 4)
	tags[0] = "正餐"
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅", TypeIDs: typeIDs, Tags: tags},
	}
	result := llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "r1", TypeIDs: []string{"type-sichuan"}, Tags: []string{"重口味"}},
		},
	}

	merged, _ := MergeLLMEnhancements(restaurants, result)

	if !slices.Equal(restaurants[0].TypeIDs, []string{"type-hotpot"}) {
		t.Fatalf("input TypeIDs mutated: %#v", restaurants[0].TypeIDs)
	}
	if !slices.Equal(restaurants[0].Tags, []string{"正餐"}) {
		t.Fatalf("input Tags mutated: %#v", restaurants[0].Tags)
	}
	if !slices.Equal(merged[0].TypeIDs, []string{"type-sichuan"}) {
		t.Fatalf("merged TypeIDs = %#v", merged[0].TypeIDs)
	}
	assertTagsEqual(t, merged[0].Tags, []string{"正餐", "重口味"})
}

func TestMergeLLMEnhancementsBuildsMixedFoodTypesFromMergedRestaurants(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅", DistanceMeters: 600, Rating: 4.6, AvgPriceCNY: 80, TypeIDs: []string{"type-hotpot"}, Tags: []string{"正餐"}},
		{ID: "r2", Name: "川味小馆", DistanceMeters: 300, Rating: 4.2, AvgPriceCNY: 60, TypeIDs: []string{"type-sichuan"}, Tags: []string{"重口味"}},
		{ID: "r3", Name: "好味道", DistanceMeters: 900, TypeIDs: []string{"type-other"}, Tags: []string{"正餐"}},
	}
	result := llm.EnhancementResult{
		Restaurants: []llm.RestaurantEnhancement{
			{ID: "r1", TypeIDs: []string{"type-sichuan"}, Tags: []string{"重口味"}},
		},
	}

	_, types := MergeLLMEnhancements(restaurants, result)

	sichuan := findType(t, types, "type-sichuan")
	if sichuan.Label != "川菜" {
		t.Fatalf("sichuan label = %q", sichuan.Label)
	}
	if sichuan.Source != "mixed" {
		t.Fatalf("sichuan source = %q", sichuan.Source)
	}
	if !slices.Equal(sichuan.RestaurantIDs, []string{"r1", "r2"}) {
		t.Fatalf("sichuan restaurant IDs = %#v", sichuan.RestaurantIDs)
	}
	if sichuan.Stats.Count != 2 || sichuan.Stats.NearestMeters != 300 || sichuan.Stats.AvgRating != 4.4 || sichuan.Stats.AvgPriceCNY != 70 {
		t.Fatalf("sichuan stats = %#v", sichuan.Stats)
	}
	assertTagsEqual(t, sichuan.Tags, []string{"正餐", "重口味"})

	other := findType(t, types, "type-other")
	if other.Label != "其他好吃的" {
		t.Fatalf("other label = %q", other.Label)
	}
}

func assertTagsEqual(t *testing.T, got []string, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("tags = %#v, want %#v", got, want)
	}
}
