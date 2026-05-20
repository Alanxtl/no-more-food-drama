"use client";

import { MapPin, Search } from "lucide-react";

type SearchInput = {
  lat: number;
  lng: number;
  radiusKm: number;
  limit: number;
};

type SearchSetupProps = {
  onSearch: (input: SearchInput) => Promise<void> | void;
};

const testLocation: SearchInput = {
  lat: 23.09,
  lng: 113.32,
  radiusKm: 3,
  limit: 20,
};

export default function SearchSetup({ onSearch }: SearchSetupProps) {
  function searchFromTestLocation() {
    void onSearch(testLocation);
  }

  return (
    <section className="mx-auto w-full max-w-md px-5 py-8">
      <h1 className="text-3xl font-bold leading-tight text-ink">先找附近能吃的</h1>
      <p className="mt-2 text-sm leading-6 text-neutral-600">
        定位失败时可以先用当前城市商圈或测试位置继续。
      </p>

      <div className="mt-6 space-y-3">
        <button
          onClick={searchFromTestLocation}
          className="inline-flex h-12 w-full items-center justify-center gap-2 rounded-md bg-accent px-4 font-semibold text-white transition hover:brightness-95 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:ring-offset-2 focus:ring-offset-paper"
          type="button"
        >
          <MapPin aria-hidden="true" className="h-[18px] w-[18px]" />
          使用测试位置搜索
        </button>
        <button
          onClick={searchFromTestLocation}
          className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-md border border-line bg-white px-4 font-semibold text-ink transition hover:border-accent/40 focus:outline-none focus:ring-2 focus:ring-accent/20 focus:ring-offset-2 focus:ring-offset-paper"
          type="button"
        >
          <Search aria-hidden="true" className="h-[18px] w-[18px]" />
          手动地址兜底
        </button>
      </div>
    </section>
  );
}
