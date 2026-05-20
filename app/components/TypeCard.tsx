"use client";

import type { FoodType, Restaurant, TypeVote } from "@/app/lib/types";

type TypeCardProps = {
  foodType: FoodType;
  restaurants: Restaurant[];
  onVote: (typeId: string, vote: TypeVote) => Promise<void> | void;
};

export default function TypeCard({ foodType, restaurants, onVote }: TypeCardProps) {
  const preview = restaurants
    .filter((restaurant) => foodType.restaurantIds.includes(restaurant.id))
    .slice(0, 3);

  return (
    <section className="mx-auto flex min-h-screen w-full max-w-md flex-col justify-center px-5 py-8">
      <div className="rounded-lg border border-line bg-white/80 p-5 shadow-sm">
        <p className="text-sm text-neutral-600">
          {foodType.stats.count} 家候选 · 最近 {foodType.stats.nearestMeters}m
        </p>
        <h1 className="mt-2 break-words text-5xl font-bold leading-tight text-ink">{foodType.label}</h1>

        {foodType.tags.length > 0 ? (
          <div className="mt-4 flex flex-wrap gap-2">
            {foodType.tags.map((tag) => (
              <span key={tag} className="rounded-full bg-paper px-3 py-1 text-xs text-neutral-700">
                {tag}
              </span>
            ))}
          </div>
        ) : null}

        {preview.length > 0 ? (
          <div className="mt-5 divide-y divide-line border-y border-line">
            {preview.map((restaurant) => (
              <div key={restaurant.id} className="py-3">
                <p className="font-semibold text-ink">{restaurant.name}</p>
                <p className="mt-1 text-sm text-neutral-600">
                  {restaurant.distanceMeters}m · {restaurant.rating ? restaurant.rating.toFixed(1) : "暂无评分"}
                </p>
              </div>
            ))}
          </div>
        ) : null}
      </div>

      <div className="mt-4 grid grid-cols-3 gap-2">
        <button
          onClick={() => void onVote(foodType.id, "avoid")}
          className="h-12 rounded-md border border-red-200 bg-white px-2 text-sm font-semibold text-danger transition hover:bg-red-50 focus:outline-none focus:ring-2 focus:ring-danger/20 focus:ring-offset-2 focus:ring-offset-paper"
          type="button"
        >
          今天不吃
        </button>
        <button
          onClick={() => void onVote(foodType.id, "neutral")}
          className="h-12 rounded-md border border-line bg-white px-2 text-sm font-semibold text-ink transition hover:border-accent/40 focus:outline-none focus:ring-2 focus:ring-accent/20 focus:ring-offset-2 focus:ring-offset-paper"
          type="button"
        >
          无所谓
        </button>
        <button
          onClick={() => void onVote(foodType.id, "want")}
          className="h-12 rounded-md bg-accent px-2 text-sm font-semibold text-white transition hover:brightness-95 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:ring-offset-2 focus:ring-offset-paper"
          type="button"
        >
          可以吃
        </button>
      </div>
    </section>
  );
}
