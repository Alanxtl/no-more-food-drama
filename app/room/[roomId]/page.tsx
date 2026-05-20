"use client";

import { useParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import ResultsList from "@/app/components/ResultsList";
import RoomLobby from "@/app/components/RoomLobby";
import SearchSetup from "@/app/components/SearchSetup";
import TypeCard from "@/app/components/TypeCard";
import {
  computeRecommendations,
  getRoom,
  joinRoom,
  overrideRestaurant,
  searchRestaurants,
  tagRoom,
  voteType,
} from "@/app/lib/api";
import { loadLlmConfig, loadParticipant, saveParticipant } from "@/app/lib/session";
import type { Room, TypeVote } from "@/app/lib/types";

export default function RoomPage() {
  const params = useParams<{ roomId: string }>();
  const roomId = params.roomId;
  const storedParticipantId = useMemo(() => {
    return typeof window === "undefined" ? null : loadParticipant(roomId);
  }, [roomId]);
  const [joinedSession, setJoinedSession] = useState<{ roomId: string; participantId: string } | null>(null);
  const [room, setRoom] = useState<Room | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [retryKey, setRetryKey] = useState(0);
  const joinedParticipantId = joinedSession?.roomId === roomId ? joinedSession.participantId : null;
  const participantId = joinedParticipantId ?? storedParticipantId;

  useEffect(() => {
    let active = true;

    async function loadRoom() {
      setErrorMessage(null);
      try {
        if (participantId) {
          const data = await getRoom(roomId);
          if (active) {
            setRoom(data.room);
          }
          return;
        }

        const data = await joinRoom(roomId);
        saveParticipant(roomId, data.participantId);
        if (active) {
          setJoinedSession({ roomId, participantId: data.participantId });
          setRoom(data.room);
        }
      } catch (error) {
        if (active) {
          setErrorMessage(errorMessageFrom(error));
        }
      }
    }

    void loadRoom();

    return () => {
      active = false;
    };
  }, [participantId, retryKey, roomId]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      getRoom(roomId)
        .then((data) => setRoom(data.room))
        .catch(() => {});
    }, 2000);

    return () => window.clearInterval(interval);
  }, [roomId]);

  const partnerOnline = useMemo(() => {
    return room ? Object.keys(room.participants).length > 1 : false;
  }, [room]);

  if (!room || !participantId) {
    return (
      <RoomMessage
        message={errorMessage ?? "加入房间中..."}
        onRetry={errorMessage ? () => setRetryKey((current) => current + 1) : undefined}
      />
    );
  }

  if (room.restaurants.length === 0) {
    return (
      <main className="min-h-screen bg-paper text-ink">
        <RoomLobby roomId={room.id} shareUrl={room.shareUrl} partnerOnline={partnerOnline} />
        <SearchSetup
          onSearch={async (input) => {
            try {
              setErrorMessage(null);
              const searched = await searchRestaurants(roomId, input);
              setRoom(searched.room);

              const llmConfig = loadLlmConfig();
              if (llmConfig) {
                tagRoom(roomId, llmConfig)
                  .then((tagged) => setRoom(tagged.room))
                  .catch(() => {});
              }
            } catch (error) {
              setErrorMessage(errorMessageFrom(error));
            }
          }}
        />
        {errorMessage ? <InlineError message={errorMessage} /> : null}
      </main>
    );
  }

  if (room.status === "results") {
    return (
      <main className="min-h-screen bg-paper text-ink">
        <ResultsList
          restaurants={room.restaurants}
          recommendations={room.recommendations}
          onRemove={async (restaurantId) => {
            try {
              setErrorMessage(null);
              const updated = await overrideRestaurant(roomId, {
                participantId,
                restaurantId,
                override: "remove",
              });
              const recomputed = await computeRecommendations(roomId);
              setRoom(recomputed.room ?? updated.room);
            } catch (error) {
              setErrorMessage(errorMessageFrom(error));
            }
          }}
        />
        {errorMessage ? <InlineError message={errorMessage} /> : null}
      </main>
    );
  }

  const foodType = nextFoodTypeForParticipant(room, participantId);

  if (!foodType) {
    return <RoomMessage message={errorMessage ?? "等另一位也筛完"} />;
  }

  return (
    <main className="min-h-screen bg-paper text-ink">
      <TypeCard
        foodType={foodType}
        restaurants={room.restaurants}
        onVote={async (typeId: string, vote: TypeVote) => {
          try {
            setErrorMessage(null);
            const updated = await voteType(roomId, { participantId, typeId, vote });
            setRoom(updated.room);

            if (allParticipantsFinished(updated.room)) {
              setRoom((await computeRecommendations(roomId)).room);
            }
          } catch (error) {
            setErrorMessage(errorMessageFrom(error));
          }
        }}
      />
      {errorMessage ? <InlineError message={errorMessage} /> : null}
    </main>
  );
}

function nextFoodTypeForParticipant(room: Room, participantId: string) {
  const votes = room.participants[participantId]?.typeVotes ?? {};
  return room.types.find((foodType) => !votes[foodType.id]);
}

function allParticipantsFinished(room: Room) {
  const participants = Object.values(room.participants);
  if (participants.length < 2 || room.types.length === 0) {
    return false;
  }
  return participants.every((participant) => room.types.every((foodType) => participant.typeVotes?.[foodType.id]));
}

function errorMessageFrom(error: unknown) {
  return error instanceof Error ? error.message : "请求失败";
}

function RoomMessage({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <main className="flex min-h-screen items-center justify-center bg-paper px-5 py-8 text-ink">
      <div className="w-full max-w-md rounded-lg border border-line bg-white/80 p-5 shadow-sm">
        <p className="text-base font-semibold">{message}</p>
        {onRetry ? (
          <button
            type="button"
            onClick={onRetry}
            className="mt-4 inline-flex h-10 items-center rounded-md bg-accent px-4 text-sm font-semibold text-white"
          >
            重试
          </button>
        ) : null}
      </div>
    </main>
  );
}

function InlineError({ message }: { message: string }) {
  return (
    <div className="mx-auto w-full max-w-md px-5 pb-8">
      <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-danger">{message}</p>
    </div>
  );
}
