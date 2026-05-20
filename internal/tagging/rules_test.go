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

	if len(types) != 3 {
		t.Fatalf("types length = %d", len(types))
	}
	if types[0].Stats.NearestMeters != 650 && types[1].Stats.NearestMeters != 650 && types[2].Stats.NearestMeters != 650 {
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
