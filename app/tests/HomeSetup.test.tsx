import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import HomeSetup from "@/app/components/HomeSetup";

describe("HomeSetup", () => {
  it("saves LLM config and creates a room", async () => {
    const onCreateRoom = vi.fn(async () => {});
    render(<HomeSetup onCreateRoom={onCreateRoom} />);

    await userEvent.type(screen.getByLabelText("API Key"), "  sk-test  ");
    await userEvent.type(screen.getByLabelText("Base URL"), "  https://api.example.com/v1  ");
    await userEvent.type(screen.getByLabelText("Model"), "  deepseek-chat  ");
    await userEvent.click(screen.getByRole("button", { name: "创建双人房间" }));

    expect(onCreateRoom).toHaveBeenCalledWith({
      apiKey: "sk-test",
      baseUrl: "https://api.example.com/v1",
      model: "deepseek-chat",
    });
  });
});
