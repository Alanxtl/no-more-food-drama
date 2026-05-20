import type { Room, TypeVote, RestaurantOverride } from "./types";

type ApiEnvelope<T> = {
  ok: boolean;
  data: T | null;
  error: null | { code: string; message: string };
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    }
  });
  const envelope = (await response.json()) as ApiEnvelope<T>;
  if (!response.ok || !envelope.ok || envelope.data == null) {
    throw new Error(envelope.error?.message ?? "请求失败");
  }
  return envelope.data;
}

export function createRoom() {
  return request<{ roomId: string; participantId: string; shareUrl: string; room: Room }>("/api/rooms", {
    method: "POST"
  });
}

export function joinRoom(roomId: string) {
  return request<{ participantId: string; room: Room }>(`/api/rooms/${roomId}/join`, {
    method: "POST"
  });
}

export function getRoom(roomId: string) {
  return request<{ room: Room }>(`/api/rooms/${roomId}`);
}

export function searchRestaurants(roomId: string, input: { lat: number; lng: number; radiusKm: number; limit: number }) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/search`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function voteType(roomId: string, input: { participantId: string; typeId: string; vote: TypeVote }) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/votes/type`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function overrideRestaurant(roomId: string, input: { participantId: string; restaurantId: string; override: RestaurantOverride }) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/votes/restaurant`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export function computeRecommendations(roomId: string) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/recommendations`, {
    method: "POST"
  });
}

export function tagRoom(roomId: string, input: { apiKey: string; baseUrl: string; model: string }) {
  return request<{ room: Room }>(`/api/rooms/${roomId}/tag`, {
    method: "POST",
    body: JSON.stringify(input)
  });
}
