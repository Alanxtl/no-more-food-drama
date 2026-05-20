package recommend

import (
	"math"
	"slices"
	"testing"

	"github.com/Alanxtl/no-more-food-drama/internal/domain"
)

func TestComputeRanksMutualWantAboveSoftAvoid(t *testing.T) {
	open := true
	room := domain.Room{
		Participants: map[string]domain.Participant{
			"p1": {TypeVotes: map[string]domain.TypeVote{"type-japanese": domain.VoteWant, "type-hotpot": domain.VoteAvoid}, RestaurantOverrides: map[string]domain.RestaurantOverride{}},
			"p2": {TypeVotes: map[string]domain.TypeVote{"type-japanese": domain.VoteWant, "type-hotpot": domain.VoteNeutral}, RestaurantOverrides: map[string]domain.RestaurantOverride{}},
		},
		Restaurants: []domain.Restaurant{
			{ID: "sushi", Name: "鮨小野", DistanceMeters: 650, Rating: 4.7, OpenNow: &open, TypeIDs: []string{"type-japanese"}, Tags: []string{"约会友好"}},
			{ID: "hotpot", Name: "热辣火锅", DistanceMeters: 300, Rating: 4.8, OpenNow: &open, TypeIDs: []string{"type-hotpot"}, Tags: []string{"重口味"}},
		},
	}

	recs := Compute(room, 5)

	if len(recs) != 2 {
		t.Fatalf("recommendations length = %d", len(recs))
	}
	if recs[0].RestaurantID != "sushi" {
		t.Fatalf("first recommendation = %s", recs[0].RestaurantID)
	}
	if recs[1].Score >= recs[0].Score {
		t.Fatalf("soft avoid should rank below mutual want: %#v", recs)
	}
	if !slices.Contains(recs[0].Reasons, "你们都点了可以吃") {
		t.Fatalf("mutual want reason missing: %#v", recs[0].Reasons)
	}
	if !slices.Contains(recs[1].Reasons, "有一位今天不太想吃这个类型") {
		t.Fatalf("soft avoid reason missing: %#v", recs[1].Reasons)
	}
}

func TestComputeHardRemovesSingleRestaurant(t *testing.T) {
	open := true
	room := domain.Room{
		Participants: map[string]domain.Participant{
			"p1": {TypeVotes: map[string]domain.TypeVote{}, RestaurantOverrides: map[string]domain.RestaurantOverride{"hotpot": domain.RestaurantRemove}},
			"p2": {TypeVotes: map[string]domain.TypeVote{}, RestaurantOverrides: map[string]domain.RestaurantOverride{}},
		},
		Restaurants: []domain.Restaurant{
			{ID: "hotpot", Name: "热辣火锅", DistanceMeters: 300, Rating: 4.8, OpenNow: &open, TypeIDs: []string{"type-hotpot"}},
			{ID: "noodle", Name: "老街米线", DistanceMeters: 900, Rating: 4.1, OpenNow: &open, TypeIDs: []string{"type-noodles"}},
		},
	}

	recs := Compute(room, 5)

	if len(recs) != 1 {
		t.Fatalf("recommendations length = %d", len(recs))
	}
	if recs[0].RestaurantID != "noodle" {
		t.Fatalf("remaining recommendation = %s", recs[0].RestaurantID)
	}
}

func TestComputeAppliesLimitRanksAndTieBreaker(t *testing.T) {
	open := true
	room := domain.Room{
		Restaurants: []domain.Restaurant{
			{ID: "b", Name: "B", DistanceMeters: 600, Rating: 4, OpenNow: &open},
			{ID: "a", Name: "A", DistanceMeters: 600, Rating: 4, OpenNow: &open},
			{ID: "c", Name: "C", DistanceMeters: 1200, Rating: 4, OpenNow: &open},
		},
	}

	recs := Compute(room, 2)

	if len(recs) != 2 {
		t.Fatalf("recommendations length = %d", len(recs))
	}
	if recs[0].RestaurantID != "a" || recs[1].RestaurantID != "b" {
		t.Fatalf("tie breaker or limit mismatch: %#v", recs)
	}
	if recs[0].Rank != 1 || recs[1].Rank != 2 {
		t.Fatalf("ranks = %d, %d", recs[0].Rank, recs[1].Rank)
	}
}

func TestComputeKeepsClosedRestaurantWithWarning(t *testing.T) {
	closed := false
	room := domain.Room{
		Restaurants: []domain.Restaurant{
			{ID: "late", Name: "夜宵", DistanceMeters: 500, Rating: 4.2, OpenNow: &closed},
		},
	}

	recs := Compute(room, 0)

	if len(recs) != 1 {
		t.Fatalf("recommendations length = %d", len(recs))
	}
	if !slices.Contains(recs[0].Warnings, "可能已经打烊") {
		t.Fatalf("closed warning missing: %#v", recs[0].Warnings)
	}
}

func TestComputeCountsMultiTypeWantOncePerParticipant(t *testing.T) {
	open := true
	room := domain.Room{
		Participants: map[string]domain.Participant{
			"p1": {TypeVotes: map[string]domain.TypeVote{"type-japanese": domain.VoteWant, "type-noodles": domain.VoteWant}},
			"p2": {TypeVotes: map[string]domain.TypeVote{}},
		},
		Restaurants: []domain.Restaurant{
			{ID: "combo", Name: "日式拉面", DistanceMeters: 600, Rating: 4, OpenNow: &open, TypeIDs: []string{"type-japanese", "type-noodles"}},
			{ID: "single", Name: "寿司", DistanceMeters: 600, Rating: 4, OpenNow: &open, TypeIDs: []string{"type-japanese"}},
		},
	}

	recs := Compute(room, 0)
	combo := recommendationByID(t, recs, "combo")
	single := recommendationByID(t, recs, "single")

	if slices.Contains(combo.Reasons, "你们都点了可以吃") {
		t.Fatalf("single participant triggered mutual want reason: %#v", combo.Reasons)
	}
	if !slices.Contains(combo.Reasons, "有人点了可以吃，另一位没有排除") {
		t.Fatalf("one-sided want reason missing: %#v", combo.Reasons)
	}
	if !almostEqual(combo.Score, single.Score) {
		t.Fatalf("multi-type want score = %.2f, single-type want score = %.2f", combo.Score, single.Score)
	}
}

func TestComputeCountsMultiTypeAvoidOncePerParticipant(t *testing.T) {
	open := true
	room := domain.Room{
		Participants: map[string]domain.Participant{
			"p1": {TypeVotes: map[string]domain.TypeVote{"type-japanese": domain.VoteAvoid, "type-noodles": domain.VoteAvoid}},
			"p2": {TypeVotes: map[string]domain.TypeVote{}},
		},
		Restaurants: []domain.Restaurant{
			{ID: "combo", Name: "日式拉面", DistanceMeters: 300, Rating: 5, OpenNow: &open, TypeIDs: []string{"type-japanese", "type-noodles"}},
			{ID: "single", Name: "寿司", DistanceMeters: 300, Rating: 5, OpenNow: &open, TypeIDs: []string{"type-japanese"}},
		},
	}

	recs := Compute(room, 0)
	combo := recommendationByID(t, recs, "combo")
	single := recommendationByID(t, recs, "single")

	if slices.Contains(combo.Reasons, "你们都不太想吃这个类型，作为备选保留") {
		t.Fatalf("single participant triggered mutual avoid reason: %#v", combo.Reasons)
	}
	if !slices.Contains(combo.Reasons, "有一位今天不太想吃这个类型") {
		t.Fatalf("one-sided avoid reason missing: %#v", combo.Reasons)
	}
	if !almostEqual(combo.Score, single.Score) {
		t.Fatalf("multi-type avoid score = %.2f, single-type avoid score = %.2f", combo.Score, single.Score)
	}
}

func TestComputeHandlesNilParticipantMaps(t *testing.T) {
	open := true
	room := domain.Room{
		Participants: map[string]domain.Participant{
			"p1": {},
		},
		Restaurants: []domain.Restaurant{
			{ID: "open", Name: "街角小馆", DistanceMeters: 500, Rating: 4, OpenNow: &open, TypeIDs: []string{"type-noodles"}},
		},
	}

	recs := Compute(room, 0)

	if len(recs) != 1 {
		t.Fatalf("recommendations length = %d", len(recs))
	}
	if recs[0].RestaurantID != "open" {
		t.Fatalf("recommendation = %s", recs[0].RestaurantID)
	}

	room.Participants = nil
	recs = Compute(room, 0)

	if len(recs) != 1 {
		t.Fatalf("recommendations length with nil participants = %d", len(recs))
	}
}

func recommendationByID(t *testing.T, recs []domain.Recommendation, restaurantID string) domain.Recommendation {
	t.Helper()
	for _, rec := range recs {
		if rec.RestaurantID == restaurantID {
			return rec
		}
	}
	t.Fatalf("recommendation %q not found in %#v", restaurantID, recs)
	return domain.Recommendation{}
}

func almostEqual(a float64, b float64) bool {
	return math.Abs(a-b) < 0.0001
}
