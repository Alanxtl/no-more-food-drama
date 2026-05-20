import type { LlmConfig } from "./types";

const LLM_CONFIG_KEY = "llmConfig";
const PARTICIPANT_KEY_PREFIX = "participant:";

export function saveLlmConfig(config: LlmConfig) {
  sessionStorage.setItem(LLM_CONFIG_KEY, JSON.stringify(config));
}

export function loadLlmConfig(): LlmConfig | null {
  const raw = sessionStorage.getItem(LLM_CONFIG_KEY);
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as LlmConfig;
  } catch {
    return null;
  }
}

export function saveParticipant(roomId: string, participantId: string) {
  sessionStorage.setItem(PARTICIPANT_KEY_PREFIX + roomId, participantId);
}

export function loadParticipant(roomId: string): string | null {
  return sessionStorage.getItem(PARTICIPANT_KEY_PREFIX + roomId);
}
