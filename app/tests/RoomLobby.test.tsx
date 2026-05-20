import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import RoomLobby from "@/app/components/RoomLobby";

describe("RoomLobby", () => {
  it("shows room code and share url", () => {
    render(<RoomLobby roomId="ABC123" shareUrl="https://app.test/room/ABC123" partnerOnline={false} />);

    expect(screen.getByText("ABC123")).toBeInTheDocument();
    expect(screen.getByText("https://app.test/room/ABC123")).toBeInTheDocument();
    expect(screen.getByText("等待另一位加入")).toBeInTheDocument();
  });
});
