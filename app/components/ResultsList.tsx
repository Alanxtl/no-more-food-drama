"use client";

import { X } from "lucide-react";
import type { Recommendation, Restaurant } from "@/app/lib/types";

type ResultsListProps = {
  restaurants: Restaurant[];
  recommendations: Recommendation[];
  onRemove: (restaurantId: string) => Promise<void> | void;
};

export default function ResultsList({ restaurants, recommendations, onRemove }: ResultsListProps) {
  const restaurantsByID = new Map(restaurants.map((restaurant) => [restaurant.id, restaurant]));

  return (
    <section className="mx-auto w-full max-w-md px-5 py-8">
      <h1 className="text-3xl font-bold leading-tight text-ink">现在就去这几家</h1>

      {recommendations.length === 0 ? (
        <div className="mt-5 rounded-lg border border-line bg-white/80 p-5 shadow-sm">
          <p className="text-base font-semibold text-ink">这轮没有共同可接受的餐厅</p>
          <p className="mt-2 text-sm leading-6 text-neutral-600">可以放宽一个类型，或者重新搜一圈附近的选择。</p>
        </div>
      ) : null}

      <div className="mt-5 space-y-3">
        {recommendations.map((recommendation) => {
          const restaurant = restaurantsByID.get(recommendation.restaurantId);
          if (!restaurant) {
            return null;
          }

          return (
            <article
              key={recommendation.restaurantId}
              className="rounded-lg border border-line bg-white/80 p-4 shadow-sm"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h2 className="break-words text-xl font-bold text-ink">
                    {recommendation.rank}. {restaurant.name}
                  </h2>
                  <p className="mt-1 text-sm text-neutral-600">
                    {restaurant.distanceMeters}m · {restaurant.rating ? restaurant.rating.toFixed(1) : "暂无评分"}
                  </p>
                </div>
                <button
                  aria-label={`剔除 ${restaurant.name}`}
                  onClick={() => void onRemove(restaurant.id)}
                  className="grid h-9 w-9 shrink-0 place-items-center rounded-md border border-line text-neutral-700 transition hover:border-danger/40 hover:text-danger focus:outline-none focus:ring-2 focus:ring-danger/20 focus:ring-offset-2 focus:ring-offset-paper"
                  type="button"
                >
                  <X aria-hidden="true" className="h-4 w-4" />
                </button>
              </div>

              <div className="mt-3 space-y-1 text-sm leading-6">
                {recommendation.reasons.map((reason) => (
                  <p key={reason} className="text-neutral-700">
                    {reason}
                  </p>
                ))}
                {recommendation.warnings.map((warning) => (
                  <p key={warning} className="text-danger">
                    {warning}
                  </p>
                ))}
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
