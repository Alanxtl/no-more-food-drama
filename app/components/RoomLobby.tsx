"use client";

import { Copy, QrCode } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";

type RoomLobbyProps = {
  roomId: string;
  shareUrl: string;
  partnerOnline: boolean;
};

export default function RoomLobby({ roomId, shareUrl, partnerOnline }: RoomLobbyProps) {
  async function copyShareUrl() {
    await navigator.clipboard?.writeText(shareUrl);
  }

  return (
    <section className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center px-5 py-8">
      <div className="space-y-3">
        <p className="text-sm font-medium text-accent">房间码</p>
        <h1 className="break-all text-4xl font-bold leading-tight text-ink">{roomId}</h1>
        <p className="text-base leading-7 text-neutral-700">
          {partnerOnline ? "另一位已加入" : "等待另一位加入"}
        </p>
      </div>

      <div className="mt-8 rounded-lg border border-line bg-white/70 p-5 shadow-sm">
        <div className="mb-5 flex items-center gap-2 text-sm font-semibold text-ink">
          <QrCode aria-hidden="true" className="h-4 w-4 text-accent" />
          <span>分享房间</span>
        </div>

        <div className="flex justify-center rounded-md border border-line bg-paper p-4">
          <QRCodeSVG value={shareUrl} size={176} />
        </div>

        <p className="mt-4 break-all rounded-md border border-line bg-paper px-3 py-2.5 text-sm text-neutral-700">
          {shareUrl}
        </p>

        <button
          type="button"
          onClick={copyShareUrl}
          className="mt-4 flex w-full items-center justify-center gap-2 rounded-md bg-accent px-4 py-3 text-sm font-semibold text-white transition hover:brightness-95 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:ring-offset-2 focus:ring-offset-paper"
        >
          <Copy aria-hidden="true" className="h-4 w-4" />
          复制链接
        </button>
      </div>
    </section>
  );
}
