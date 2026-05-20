import { describe, expect, it, beforeEach } from "vitest";
import { loadLlmConfig, saveLlmConfig } from "@/app/lib/session";

describe("LLM config session storage", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it("stores LLM config only in sessionStorage", () => {
    saveLlmConfig({
      apiKey: "sk-test",
      baseUrl: "https://api.example.com/v1",
      model: "deepseek-chat"
    });

    expect(loadLlmConfig()).toEqual({
      apiKey: "sk-test",
      baseUrl: "https://api.example.com/v1",
      model: "deepseek-chat"
    });
    expect(localStorage.getItem("llmConfig")).toBeNull();
  });

  it("returns null for malformed stored LLM config", () => {
    sessionStorage.setItem("llmConfig", "not-json");

    expect(loadLlmConfig()).toBeNull();
  });
});
