package tagging

import (
	"slices"
	"strings"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
	"github.com/Alanxtl/no-more-food-drama/internal/llm"
)

func MergeLLMEnhancements(restaurants []domain.Restaurant, result llm.EnhancementResult) ([]domain.Restaurant, []domain.FoodType) {
	enhancementsByID := make(map[string]llm.RestaurantEnhancement, len(result.Restaurants))
	for _, enhancement := range result.Restaurants {
		enhancementsByID[enhancement.ID] = enhancement
	}

	merged := make([]domain.Restaurant, len(restaurants))
	copy(merged, restaurants)
	typeMap := map[string]*foodTypeAggregate{}

	for i := range merged {
		merged[i].TypeIDs = slices.Clone(merged[i].TypeIDs)
		merged[i].Tags = slices.Clone(merged[i].Tags)

		if enhancement, ok := enhancementsByID[merged[i].ID]; ok {
			typeIDs := cleanUnique(enhancement.TypeIDs)
			if len(typeIDs) > 0 {
				merged[i].TypeIDs = typeIDs
			}
			for _, tag := range cleanUnique(enhancement.Tags) {
				merged[i].Tags = appendUnique(merged[i].Tags, tag)
			}
		}

		merged[i].TypeIDs = cleanUnique(merged[i].TypeIDs)
		for _, typeID := range merged[i].TypeIDs {
			addToType(typeMap, rule{ID: typeID, Label: labelForType(typeID), Tags: merged[i].Tags}, merged[i])
			typeMap[typeID].foodType.Source = "mixed"
		}
	}

	return merged, flattenTypes(typeMap)
}

func cleanUnique(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = appendUnique(cleaned, value)
	}
	return cleaned
}

func labelForType(typeID string) string {
	typeID = strings.TrimSpace(typeID)
	for _, rule := range foodRules {
		if rule.ID == typeID {
			return rule.Label
		}
	}
	if typeID == "type-other" {
		return "其他好吃的"
	}
	return strings.TrimPrefix(typeID, "type-")
}
