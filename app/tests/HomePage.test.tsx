import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import HomePage from "@/app/page";
import { createRoom } from "@/app/lib/api";

const push = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
}));

vi.mock("@/app/lib/api", () => ({
  createRoom: vi.fn(),
}));

describe("HomePage", () => {
  beforeEach(() => {
    sessionStorage.clear();
    push.mockClear();
    vi.mocked(createRoom).mockReset();
  });

  it("creates a room before saving session state and routing", async () => {
    vi.mocked(createRoom).mockResolvedValue({
      roomId: "ABC123",
      participantId: "p1",
      shareUrl: "https://app.test/room/ABC123",
      room: {} as never,
    });

    render(<HomePage />);

    await userEvent.type(screen.getByLabelText("API Key"), "sk-test");
    await userEvent.type(screen.getByLabelText("Base URL"), "https://api.example.com/v1");
    await userEvent.type(screen.getByLabelText("Model"), "deepseek-chat");
    await userEvent.click(screen.getByRole("button", { name: "创建双人房间" }));

    await waitFor(() => expect(push).toHaveBeenCalledWith("/room/ABC123"));
    expect(sessionStorage.getItem("participant:ABC123")).toBe("p1");
    expect(sessionStorage.getItem("llmConfig")).toContain("sk-test");
  });

  it("does not save LLM config when room creation fails", async () => {
    vi.mocked(createRoom).mockRejectedValue(new Error("create failed"));

    render(<HomePage />);

    await userEvent.type(screen.getByLabelText("API Key"), "sk-test");
    await userEvent.type(screen.getByLabelText("Base URL"), "https://api.example.com/v1");
    await userEvent.type(screen.getByLabelText("Model"), "deepseek-chat");
    await userEvent.click(screen.getByRole("button", { name: "创建双人房间" }));

    expect(await screen.findByText("create failed")).toBeInTheDocument();
    expect(sessionStorage.getItem("llmConfig")).toBeNull();
    expect(push).not.toHaveBeenCalled();
  });
});
