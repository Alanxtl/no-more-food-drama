import { expect, test } from "@playwright/test";

test("two people can create a room, search, vote, and see recommendations", async ({ browser }) => {
  const creator = await browser.newPage();
  await creator.goto("/");
  await creator.getByRole("button", { name: "创建双人房间" }).click();
  await expect(creator.getByText("房间码")).toBeVisible();

  const roomUrl = creator.url();
  const partner = await browser.newPage();
  await partner.goto(roomUrl);
  await expect(partner.getByText("另一位已加入")).toBeVisible();

  await creator.getByRole("button", { name: "使用测试位置搜索" }).click();
  await expect(creator.getByText("日料")).toBeVisible();

  await creator.getByRole("button", { name: "可以吃" }).click();
  await expect(creator.getByRole("heading", { name: "火锅" })).toBeVisible();
  await creator.getByRole("button", { name: "无所谓" }).click();
  await expect(creator.getByText("等另一位也筛完")).toBeVisible();

  await partner.reload();
  await expect(partner.getByText("日料")).toBeVisible();
  await partner.getByRole("button", { name: "可以吃" }).click();
  await expect(partner.getByRole("heading", { name: "火锅" })).toBeVisible();
  await partner.getByRole("button", { name: "无所谓" }).click();

  await expect(creator.getByText("现在就去这几家")).toBeVisible({ timeout: 10_000 });
});
