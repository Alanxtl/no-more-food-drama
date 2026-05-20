package domain

import "time"

const RoomTTL = time.Hour

type RoomStatus string

const (
	StatusLobby     RoomStatus = "lobby"
	StatusSearching RoomStatus = "searching"
	StatusTagging   RoomStatus = "tagging"
	StatusFiltering RoomStatus = "filtering"
	StatusResults   RoomStatus = "results"
)

type Role string

const (
	RoleCreator Role = "creator"
	RolePartner Role = "partner"
)

type TypeVote string

const (
	VoteWant    TypeVote = "want"
	VoteNeutral TypeVote = "neutral"
	VoteAvoid   TypeVote = "avoid"
)

type RestaurantOverride string

const (
	RestaurantKeep   RestaurantOverride = "keep"
	RestaurantRemove RestaurantOverride = "remove"
)

type Room struct {
	ID              string                 `json:"id"`
	Version         int                    `json:"version"`
	ShareURL        string                 `json:"shareUrl"`
	CreatedAt       time.Time              `json:"createdAt"`
	ExpiresAt       time.Time              `json:"expiresAt"`
	Status          RoomStatus             `json:"status"`
	SearchConfig    *SearchConfig          `json:"searchConfig,omitempty"`
	Participants    map[string]Participant `json:"participants"`
	Restaurants     []Restaurant           `json:"restaurants"`
	Types           []FoodType             `json:"types"`
	Recommendations []Recommendation       `json:"recommendations"`
}

type SearchConfig struct {
	LocationText string  `json:"locationText,omitempty"`
	Lat          float64 `json:"lat,omitempty"`
	Lng          float64 `json:"lng,omitempty"`
	RadiusKM     int     `json:"radiusKm"`
	Limit        int     `json:"limit"`
}

type Participant struct {
	DisplayName         string                        `json:"displayName"`
	Role                Role                          `json:"role"`
	JoinedAt            time.Time                     `json:"joinedAt"`
	LastSeenAt          time.Time                     `json:"lastSeenAt"`
	TypeVotes           map[string]TypeVote           `json:"typeVotes"`
	RestaurantOverrides map[string]RestaurantOverride `json:"restaurantOverrides"`
}

type Restaurant struct {
	ID             string   `json:"id"`
	Provider       string   `json:"provider"`
	ProviderID     string   `json:"providerId"`
	Name           string   `json:"name"`
	Address        string   `json:"address"`
	Lat            float64  `json:"lat"`
	Lng            float64  `json:"lng"`
	DistanceMeters int      `json:"distanceMeters"`
	Rating         float64  `json:"rating,omitempty"`
	PriceLevel     string   `json:"priceLevel,omitempty"`
	AvgPriceCNY    int      `json:"avgPriceCny,omitempty"`
	OpenNow        *bool    `json:"openNow,omitempty"`
	Categories     []string `json:"categories"`
	TypeIDs        []string `json:"typeIds"`
	Tags           []string `json:"tags"`
}

type FoodType struct {
	ID            string        `json:"id"`
	Label         string        `json:"label"`
	Source        string        `json:"source"`
	Tags          []string      `json:"tags"`
	RestaurantIDs []string      `json:"restaurantIds"`
	Stats         FoodTypeStats `json:"stats"`
}

type FoodTypeStats struct {
	Count         int     `json:"count"`
	NearestMeters int     `json:"nearestMeters"`
	AvgRating     float64 `json:"avgRating,omitempty"`
	AvgPriceCNY   int     `json:"avgPriceCny,omitempty"`
}

type Recommendation struct {
	RestaurantID string   `json:"restaurantId"`
	Score        float64  `json:"score"`
	Rank         int      `json:"rank"`
	Reasons      []string `json:"reasons"`
	Warnings     []string `json:"warnings"`
}
