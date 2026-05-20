package recommend

import (
	"fmt"
	"sort"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func Compute(room domain.Room, limit int) []domain.Recommendation {
	scored := make([]domain.Recommendation, 0, len(room.Restaurants))
	for _, restaurant := range room.Restaurants {
		if removedByAnyone(room, restaurant.ID) {
			continue
		}

		score, reasons, warnings := scoreRestaurant(room, restaurant)
		scored = append(scored, domain.Recommendation{
			RestaurantID: restaurant.ID,
			Score:        score,
			Reasons:      reasons,
			Warnings:     warnings,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score == scored[j].Score {
			return scored[i].RestaurantID < scored[j].RestaurantID
		}
		return scored[i].Score > scored[j].Score
	})

	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}
	for i := range scored {
		scored[i].Rank = i + 1
	}

	return scored
}

func scoreRestaurant(room domain.Room, restaurant domain.Restaurant) (float64, []string, []string) {
	score := 0.0
	reasons := []string{}
	warnings := []string{}

	if restaurant.Rating > 0 {
		score += min(restaurant.Rating/5*22, 22)
		reasons = append(reasons, fmt.Sprintf("评分 %.1f", restaurant.Rating))
	}

	if restaurant.OpenNow != nil {
		if *restaurant.OpenNow {
			score += 8
			reasons = append(reasons, "正在营业")
		} else {
			score -= 18
			warnings = append(warnings, "可能已经打烊")
		}
	}

	if restaurant.Address != "" && restaurant.DistanceMeters > 0 {
		score += 5
	}

	if restaurant.DistanceMeters > 0 {
		distanceScore := 20 - float64(restaurant.DistanceMeters)/3000*20
		score += max(0, distanceScore)
		if restaurant.DistanceMeters <= 800 {
			reasons = append(reasons, fmt.Sprintf("离你们 %dm", restaurant.DistanceMeters))
		}
		if restaurant.DistanceMeters > 2500 {
			warnings = append(warnings, "距离偏远")
		}
	}

	preferenceScore, preferenceReason := preference(room, restaurant)
	score += preferenceScore
	if preferenceReason != "" {
		reasons = append(reasons, preferenceReason)
	}

	for _, tag := range restaurant.Tags {
		switch tag {
		case "约会友好", "快速解决", "性价比高", "离得近":
			score += 2.5
		case "可能排队":
			score -= 5
			warnings = append(warnings, "可能要排队")
		}
	}

	return max(0, min(100, score)), reasons, warnings
}

func removedByAnyone(room domain.Room, restaurantID string) bool {
	for _, participant := range room.Participants {
		if participant.RestaurantOverrides[restaurantID] == domain.RestaurantRemove {
			return true
		}
	}
	return false
}

func preference(room domain.Room, restaurant domain.Restaurant) (float64, string) {
	score := 0.0
	wantCount := 0
	avoidCount := 0

	for _, participant := range room.Participants {
		wantsRestaurant := false
		avoidsRestaurant := false
		for _, typeID := range restaurant.TypeIDs {
			switch participant.TypeVotes[typeID] {
			case domain.VoteWant:
				wantsRestaurant = true
			case domain.VoteAvoid:
				avoidsRestaurant = true
			}
		}

		if wantsRestaurant {
			score += 15
			wantCount++
		} else if avoidsRestaurant {
			score -= 16
			avoidCount++
		}
	}

	if wantCount >= 2 {
		return score, "你们都点了可以吃"
	}
	if wantCount == 1 && avoidCount == 0 {
		return score, "有人点了可以吃，另一位没有排除"
	}
	if avoidCount >= 2 {
		return score, "你们都不太想吃这个类型，作为备选保留"
	}
	if avoidCount == 1 {
		return score, "有一位今天不太想吃这个类型"
	}
	return score, ""
}

func min(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
