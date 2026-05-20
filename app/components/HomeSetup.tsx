"use client";

import { KeyRound, Plus } from "lucide-react";
import type { FormEvent } from "react";
import type { LlmConfig } from "@/app/lib/types";

type HomeSetupProps = {
  onCreateRoom: (config: LlmConfig | null) => Promise<void> | void;
};

export default function HomeSetup({ onCreateRoom }: HomeSetupProps) {
  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const formData = new FormData(event.currentTarget);
    const apiKey = String(formData.get("apiKey") ?? "").trim();
    const baseUrl = String(formData.get("baseUrl") ?? "").trim();
    const model = String(formData.get("model") ?? "").trim();
    const config = apiKey && baseUrl && model ? { apiKey, baseUrl, model } : null;

    await onCreateRoom(config);
  }

  return (
    <section className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center px-5 py-8">
      <div className="space-y-3">
        <p className="text-sm font-medium text-accent">no-more-food-drama</p>
        <h1 className="text-4xl font-bold leading-tight text-ink">让你选你又不选</h1>
        <p className="text-base leading-7 text-neutral-700">
          填好模型配置，先开一个双人房间，再一起筛今天吃什么。
        </p>
      </div>

      <form
        onSubmit={handleSubmit}
        className="mt-8 rounded-lg border border-line bg-white/70 p-5 shadow-sm"
      >
        <div className="mb-5 flex items-center gap-2 text-sm font-semibold text-ink">
          <KeyRound aria-hidden="true" className="h-4 w-4 text-accent" />
          <span>LLM 配置</span>
        </div>

        <div className="space-y-4">
          <label className="block text-sm font-medium text-ink">
            API Key
            <input
              name="apiKey"
              type="password"
              autoComplete="off"
              className="mt-2 w-full rounded-md border border-line bg-paper px-3 py-2.5 text-ink outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/20"
            />
          </label>

          <label className="block text-sm font-medium text-ink">
            Base URL
            <input
              name="baseUrl"
              type="url"
              className="mt-2 w-full rounded-md border border-line bg-paper px-3 py-2.5 text-ink outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/20"
            />
          </label>

          <label className="block text-sm font-medium text-ink">
            Model
            <input
              name="model"
              type="text"
              className="mt-2 w-full rounded-md border border-line bg-paper px-3 py-2.5 text-ink outline-none transition focus:border-accent focus:ring-2 focus:ring-accent/20"
            />
          </label>
        </div>

        <button
          type="submit"
          className="mt-6 flex w-full items-center justify-center gap-2 rounded-md bg-accent px-4 py-3 text-sm font-semibold text-white transition hover:brightness-95 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:ring-offset-2 focus:ring-offset-paper"
        >
          <Plus aria-hidden="true" className="h-4 w-4" />
          创建双人房间
        </button>
      </form>
    </section>
  );
}
