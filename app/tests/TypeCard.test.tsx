import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import TypeCard from "@/app/components/TypeCard";
import type { FoodType, Restaurant } from "@/app/lib/types";

const foodType: FoodType = {
  id: "type-japanese",
  label: "日料",
  source: "rules",
  tags: ["约会友好", "清淡"],
  restaurantIds: ["r1"],
  stats: { count: 1, nearestMeters: 650, avgRating: 4.7, avgPriceCny: 128 },
};

const restaurants: Restaurant[] = [
  {
    id: "r1",
    provider: "amap",
    providerId: "r1",
    name: "鮨小野",
    address: "测试路",
    lat: 1,
    lng: 1,
    distanceMeters: 650,
    rating: 4.7,
    avgPriceCny: 128,
    categories: [],
    typeIds: ["type-japanese"],
    tags: ["约会友好"],
  },
];

describe("TypeCard", () => {
  it("votes on a food type", async () => {
    const onVote = vi.fn();
    render(<TypeCard foodType={foodType} restaurants={restaurants} onVote={onVote} />);

    expect(screen.getByText("日料")).toBeInTheDocument();
    expect(screen.getByText("鮨小野")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "今天不吃" }));

    expect(onVote).toHaveBeenCalledWith("type-japanese", "avoid");
  });
});
