package tagging

import (
	"testing"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func TestBuildRuleTagsGroupsRestaurantsIntoTypes(t *testing.T) {
	open := true
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "鮨小野", DistanceMeters: 650, Rating: 4.7, AvgPriceCNY: 128, OpenNow: &open, Categories: []string{"美食", "日本料理"}},
		{ID: "r2", Name: "热辣火锅", DistanceMeters: 900, Rating: 4.5, AvgPriceCNY: 98, OpenNow: &open, Categories: []string{"美食", "火锅"}},
		{ID: "r3", Name: "老街螺蛳粉", DistanceMeters: 300, Rating: 4.2, AvgPriceCNY: 28, OpenNow: &open, Categories: []string{"美食", "小吃快餐"}},
	}

	tagged, types := BuildRuleTags(restaurants)

	assertRestaurantHasType(t, tagged, "r1", "type-japanese")
	assertRestaurantHasType(t, tagged, "r2", "type-hotpot")
	assertRestaurantHasTag(t, tagged, "r3", "快速解决")

	if len(types) != 5 {
		t.Fatalf("types length = %d", len(types))
	}
	if types[0].Stats.NearestMeters != 650 && types[1].Stats.NearestMeters != 650 && types[2].Stats.NearestMeters != 650 && types[3].Stats.NearestMeters != 650 && types[4].Stats.NearestMeters != 650 {
		t.Fatalf("expected a type with nearest distance 650m: %#v", types)
	}
}

func TestBuildRuleTagsUsesOtherTypeForUnknownFood(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "好味道", DistanceMeters: 1000, Categories: []string{"美食"}},
	}

	tagged, types := BuildRuleTags(restaurants)

	assertRestaurantHasType(t, tagged, "r1", "type-other")
	if types[0].Label != "其他好吃的" {
		t.Fatalf("label = %q", types[0].Label)
	}
}

func TestBuildRuleTagsAllowsMultipleRuleTypes(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "东京拉面", DistanceMeters: 500, Categories: []string{"日本料理", "粉面"}},
	}

	tagged, types := BuildRuleTags(restaurants)

	assertRestaurantHasType(t, tagged, "r1", "type-japanese")
	assertRestaurantHasType(t, tagged, "r1", "type-noodles")
	assertRestaurantHasTag(t, tagged, "r1", "清淡")
	assertRestaurantHasTag(t, tagged, "r1", "快速解决")
	if len(types) != 2 {
		t.Fatalf("types length = %d", len(types))
	}
}

func TestBuildRuleTagsAveragesRatingIgnoringMissingRatings(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅一号", DistanceMeters: 500, Categories: []string{"火锅"}},
		{ID: "r2", Name: "热辣火锅二号", DistanceMeters: 700, Rating: 4.6, Categories: []string{"火锅"}},
	}

	_, types := BuildRuleTags(restaurants)

	hotpot := findType(t, types, "type-hotpot")
	if hotpot.Stats.AvgRating != 4.6 {
		t.Fatalf("AvgRating = %v", hotpot.Stats.AvgRating)
	}
}

func TestBuildRuleTagsAveragesPriceIgnoringMissingPrices(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅一号", DistanceMeters: 500, Categories: []string{"火锅"}},
		{ID: "r2", Name: "热辣火锅二号", DistanceMeters: 700, AvgPriceCNY: 80, Categories: []string{"火锅"}},
	}

	_, types := BuildRuleTags(restaurants)

	hotpot := findType(t, types, "type-hotpot")
	if hotpot.Stats.AvgPriceCNY != 80 {
		t.Fatalf("AvgPriceCNY = %v", hotpot.Stats.AvgPriceCNY)
	}
}

func TestBuildRuleTagsDoesNotMutateInputTypeIDsAndTags(t *testing.T) {
	typeIDs := make([]string, 1, 4)
	typeIDs[0] = "existing-type"
	tags := make([]string, 1, 4)
	tags[0] = "existing-tag"
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅", DistanceMeters: 500, Categories: []string{"火锅"}, TypeIDs: typeIDs, Tags: tags},
	}

	BuildRuleTags(restaurants)

	if len(restaurants[0].TypeIDs) != 1 || restaurants[0].TypeIDs[0] != "existing-type" {
		t.Fatalf("input TypeIDs mutated: %#v", restaurants[0].TypeIDs)
	}
	if len(restaurants[0].Tags) != 1 || restaurants[0].Tags[0] != "existing-tag" {
		t.Fatalf("input Tags mutated: %#v", restaurants[0].Tags)
	}
}

func TestBuildRuleTagsDoesNotTreatUnknownDistanceAsNear(t *testing.T) {
	restaurants := []domain.Restaurant{
		{ID: "r1", Name: "热辣火锅", DistanceMeters: 0, Categories: []string{"火锅"}},
	}

	tagged, _ := BuildRuleTags(restaurants)

	assertRestaurantDoesNotHaveTag(t, tagged, "r1", "离得近")
}

func assertRestaurantHasType(t *testing.T, restaurants []domain.Restaurant, restaurantID string, typeID string) {
	t.Helper()
	for _, restaurant := range restaurants {
		if restaurant.ID != restaurantID {
			continue
		}
		for _, got := range restaurant.TypeIDs {
			if got == typeID {
				return
			}
		}
		t.Fatalf("restaurant %s missing type %s: %#v", restaurantID, typeID, restaurant.TypeIDs)
	}
	t.Fatalf("restaurant %s not found", restaurantID)
}

func assertRestaurantDoesNotHaveTag(t *testing.T, restaurants []domain.Restaurant, restaurantID string, tag string) {
	t.Helper()
	for _, restaurant := range restaurants {
		if restaurant.ID != restaurantID {
			continue
		}
		for _, got := range restaurant.Tags {
			if got == tag {
				t.Fatalf("restaurant %s has unexpected tag %s: %#v", restaurantID, tag, restaurant.Tags)
			}
		}
		return
	}
	t.Fatalf("restaurant %s not found", restaurantID)
}

func findType(t *testing.T, types []domain.FoodType, typeID string) domain.FoodType {
	t.Helper()
	for _, foodType := range types {
		if foodType.ID == typeID {
			return foodType
		}
	}
	t.Fatalf("type %s not found: %#v", typeID, types)
	return domain.FoodType{}
}

func assertRestaurantHasTag(t *testing.T, restaurants []domain.Restaurant, restaurantID string, tag string) {
	t.Helper()
	for _, restaurant := range restaurants {
		if restaurant.ID != restaurantID {
			continue
		}
		for _, got := range restaurant.Tags {
			if got == tag {
				return
			}
		}
		t.Fatalf("restaurant %s missing tag %s: %#v", restaurantID, tag, restaurant.Tags)
	}
	t.Fatalf("restaurant %s not found", restaurantID)
}
