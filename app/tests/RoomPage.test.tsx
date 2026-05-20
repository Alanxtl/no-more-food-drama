import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import RoomPage from "@/app/room/[roomId]/page";
import {
  computeRecommendations,
  getRoom,
  joinRoom,
  overrideRestaurant,
  searchRestaurants,
  tagRoom,
  voteType,
} from "@/app/lib/api";
import type { Room } from "@/app/lib/types";

vi.mock("next/navigation", () => ({
  useParams: () => ({ roomId: "ABC123" }),
}));

vi.mock("@/app/lib/api", () => ({
  computeRecommendations: vi.fn(),
  getRoom: vi.fn(),
  joinRoom: vi.fn(),
  overrideRestaurant: vi.fn(),
  searchRestaurants: vi.fn(),
  tagRoom: vi.fn(),
  voteType: vi.fn(),
}));

const api = {
  computeRecommendations: vi.mocked(computeRecommendations),
  getRoom: vi.mocked(getRoom),
  joinRoom: vi.mocked(joinRoom),
  overrideRestaurant: vi.mocked(overrideRestaurant),
  searchRestaurants: vi.mocked(searchRestaurants),
  tagRoom: vi.mocked(tagRoom),
  voteType: vi.mocked(voteType),
};

function room(overrides: Partial<Room> = {}): Room {
  return {
    id: "ABC123",
    version: 1,
    shareUrl: "https://app.test/room/ABC123",
    createdAt: "2026-05-20T12:00:00Z",
    expiresAt: "2026-05-20T13:00:00Z",
    status: "filtering",
    participants: {
      p1: {
        displayName: "我",
        role: "creator",
        joinedAt: "2026-05-20T12:00:00Z",
        lastSeenAt: "2026-05-20T12:00:00Z",
        typeVotes: {},
        restaurantOverrides: {},
      },
      p2: {
        displayName: "另一位",
        role: "partner",
        joinedAt: "2026-05-20T12:00:00Z",
        lastSeenAt: "2026-05-20T12:00:00Z",
        typeVotes: {},
        restaurantOverrides: {},
      },
    },
    restaurants: [
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
        typeIds: ["type-japanese"],
        tags: [],
      },
    ],
    types: [
      {
        id: "type-japanese",
        label: "日料",
        source: "rules",
        tags: ["清淡"],
        restaurantIds: ["r1"],
        stats: { count: 1, nearestMeters: 650 },
      },
    ],
    recommendations: [],
    ...overrides,
  };
}

describe("RoomPage", () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.clearAllMocks();
    api.computeRecommendations.mockResolvedValue({ room: room({ status: "results" }) });
    api.getRoom.mockResolvedValue({ room: room() });
    api.joinRoom.mockResolvedValue({ participantId: "p1", room: room() });
    api.overrideRestaurant.mockResolvedValue({ room: room({ status: "results" }) });
    api.searchRestaurants.mockResolvedValue({ room: room() });
    api.tagRoom.mockResolvedValue({ room: room() });
    api.voteType.mockResolvedValue({
      room: room({
        participants: {
          ...room().participants,
          p1: {
            ...room().participants.p1,
            typeVotes: { "type-japanese": "want" },
          },
        },
      }),
    });
  });

  it("waits for both participants before computing recommendations", async () => {
    sessionStorage.setItem("participant:ABC123", "p1");
    render(<RoomPage />);

    await userEvent.click(await screen.findByRole("button", { name: "可以吃" }));

    await waitFor(() => expect(api.voteType).toHaveBeenCalledWith("ABC123", {
      participantId: "p1",
      typeId: "type-japanese",
      vote: "want",
    }));
    expect(api.computeRecommendations).not.toHaveBeenCalled();
    expect(await screen.findByText("等另一位也筛完")).toBeInTheDocument();
  });

  it("shows an empty result state when recommendations are empty", async () => {
    sessionStorage.setItem("participant:ABC123", "p1");
    api.getRoom.mockResolvedValue({ room: room({ status: "results", recommendations: [] }) });

    render(<RoomPage />);

    expect(await screen.findByText("这轮没有共同可接受的餐厅")).toBeInTheDocument();
    expect(screen.queryByText("日料")).not.toBeInTheDocument();
  });

  it("shows a visible error when joining the room fails", async () => {
    api.joinRoom.mockRejectedValue(new Error("room not found"));

    render(<RoomPage />);

    expect(await screen.findByText("room not found")).toBeInTheDocument();
  });

  it("shows a visible error when final recommendation computation fails", async () => {
    sessionStorage.setItem("participant:ABC123", "p1");
    api.getRoom.mockResolvedValue({
      room: room({
        participants: {
          ...room().participants,
          p2: {
            ...room().participants.p2,
            typeVotes: { "type-japanese": "neutral" },
          },
        },
      }),
    });
    api.voteType.mockResolvedValue({
      room: room({
        participants: {
          p1: {
            ...room().participants.p1,
            typeVotes: { "type-japanese": "want" },
          },
          p2: {
            ...room().participants.p2,
            typeVotes: { "type-japanese": "neutral" },
          },
        },
      }),
    });
    api.computeRecommendations.mockRejectedValue(new Error("recommendation failed"));

    render(<RoomPage />);

    await userEvent.click(await screen.findByRole("button", { name: "可以吃" }));

    expect(await screen.findByText("recommendation failed")).toBeInTheDocument();
  });
});
