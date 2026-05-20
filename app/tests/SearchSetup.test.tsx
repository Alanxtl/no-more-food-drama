import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import SearchSetup from "@/app/components/SearchSetup";

describe("SearchSetup", () => {
  it("starts a nearby restaurant search from the test location", async () => {
    const onSearch = vi.fn();
    render(<SearchSetup onSearch={onSearch} />);

    await userEvent.click(screen.getByRole("button", { name: "使用测试位置搜索" }));

    expect(onSearch).toHaveBeenCalledWith({
      lat: 23.09,
      lng: 113.32,
      radiusKm: 3,
      limit: 20,
    });
  });
});
