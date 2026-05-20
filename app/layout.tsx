import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "让你选你又不选",
  description: "双人附近餐厅决策工具"
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN">
      <body>{children}</body>
    </html>
  );
}
