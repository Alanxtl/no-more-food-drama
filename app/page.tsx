"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import HomeSetup from "./components/HomeSetup";
import { createRoom } from "./lib/api";
import { saveLlmConfig, saveParticipant } from "./lib/session";
import type { LlmConfig } from "./lib/types";

export default function HomePage() {
  const router = useRouter();
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  async function handleCreate(config: LlmConfig | null) {
    try {
      setErrorMessage(null);
      const data = await createRoom();
      saveParticipant(data.roomId, data.participantId);
      if (config) {
        saveLlmConfig(config);
      }
      router.push(`/room/${data.roomId}`);
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "创建房间失败");
    }
  }

  return (
    <main className="min-h-screen bg-paper text-ink">
      <HomeSetup onCreateRoom={handleCreate} />
      {errorMessage ? (
        <div className="mx-auto w-full max-w-md px-5 pb-8">
          <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">{errorMessage}</p>
        </div>
      ) : null}
    </main>
  );
}
