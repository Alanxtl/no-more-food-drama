import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRoom } from "@/app/lib/api";

describe("api client", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("unwraps successful API responses", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({
      ok: true,
      json: async () => ({ ok: true, data: { roomId: "ABC123" }, error: null })
    })));

    await expect(createRoom()).resolves.toEqual({ roomId: "ABC123" });
  });

  it("does not keep a stubbed fetch between tests", () => {
    expect(vi.isMockFunction(fetch)).toBe(false);
  });

  it("rejects when the HTTP response is not ok", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({
      ok: false,
      json: async () => ({ ok: true, data: { roomId: "ABC123" }, error: null })
    })));

    await expect(createRoom()).rejects.toThrow("请求失败");
  });

  it("rejects when the API envelope is not ok", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({
      ok: true,
      json: async () => ({ ok: false, data: null, error: null })
    })));

    await expect(createRoom()).rejects.toThrow("请求失败");
  });

  it("propagates backend error messages", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({
      ok: true,
      json: async () => ({ ok: false, data: null, error: { code: "NO_ROOM", message: "Room not found" } })
    })));

    await expect(createRoom()).rejects.toThrow("Room not found");
  });

  it("unwraps falsy non-null API data", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => ({
      ok: true,
      json: async () => ({ ok: true, data: "", error: null })
    })));

    await expect(createRoom()).resolves.toBe("");
  });
});
