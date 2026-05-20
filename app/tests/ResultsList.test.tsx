import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import ResultsList from "@/app/components/ResultsList";
import type { Recommendation, Restaurant } from "@/app/lib/types";

describe("ResultsList", () => {
  it("shows ranked recommendation reasons and warnings", () => {
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
        categories: [],
        typeIds: [],
        tags: [],
      },
    ];
    const recommendations: Recommendation[] = [
      {
        restaurantId: "r1",
        rank: 1,
        score: 92,
        reasons: ["离你们 650m", "正在营业"],
        warnings: ["可能要排队"],
      },
    ];

    render(<ResultsList restaurants={restaurants} recommendations={recommendations} onRemove={() => {}} />);

    expect(screen.getByText("1. 鮨小野")).toBeInTheDocument();
    expect(screen.getByText("离你们 650m")).toBeInTheDocument();
    expect(screen.getByText("可能要排队")).toBeInTheDocument();
  });
});
