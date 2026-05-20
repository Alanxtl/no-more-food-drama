export type RoomStatus = "lobby" | "searching" | "tagging" | "filtering" | "results";
export type Role = "creator" | "partner";
export type TypeVote = "want" | "neutral" | "avoid";
export type RestaurantOverride = "keep" | "remove";

export type LlmConfig = {
  apiKey: string;
  baseUrl: string;
  model: string;
};

export type Room = {
  id: string;
  version: number;
  shareUrl: string;
  createdAt: string;
  expiresAt: string;
  status: RoomStatus;
  searchConfig?: SearchConfig;
  participants: Record<string, Participant>;
  restaurants: Restaurant[];
  types: FoodType[];
  recommendations: Recommendation[];
};

export type SearchConfig = {
  locationText?: string;
  lat?: number;
  lng?: number;
  radiusKm: number;
  limit: number;
};

export type Participant = {
  displayName: string;
  role: Role;
  joinedAt: string;
  lastSeenAt: string;
  typeVotes: Record<string, TypeVote>;
  restaurantOverrides: Record<string, RestaurantOverride>;
};

export type Restaurant = {
  id: string;
  provider: "amap";
  providerId: string;
  name: string;
  address: string;
  lat: number;
  lng: number;
  distanceMeters: number;
  rating?: number;
  priceLevel?: string;
  avgPriceCny?: number;
  openNow?: boolean;
  categories: string[];
  typeIds: string[];
  tags: string[];
};

export type FoodType = {
  id: string;
  label: string;
  source: "rules" | "llm" | "mixed";
  tags: string[];
  restaurantIds: string[];
  stats: {
    count: number;
    nearestMeters: number;
    avgRating?: number;
    avgPriceCny?: number;
  };
};

export type Recommendation = {
  restaurantId: string;
  score: number;
  rank: number;
  reasons: string[];
  warnings: string[];
};
