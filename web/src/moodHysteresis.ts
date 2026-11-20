import type { Mood } from './types';

const rank: Record<Mood, number> = { calm: 0, worried: 1, panic: 2 };

export type HysteresisState = {
  displayed: Mood;
  pending?: Mood;
  pendingSince?: number;
};

export function nextMood(state: HysteresisState, raw: Mood, now: number, calmDelayMs = 120_000): HysteresisState {
  if (rank[raw] > rank[state.displayed]) {
    return { displayed: raw };
  }
  if (raw === state.displayed) {
    return { displayed: state.displayed };
  }
  if (state.pending !== raw) {
    return { displayed: state.displayed, pending: raw, pendingSince: now };
  }
  if (state.pendingSince !== undefined && now-state.pendingSince >= calmDelayMs) {
    return { displayed: raw };
  }
  return state;
}

export function foxQuote(mood: Mood): string {
  switch (mood) {
    case 'calm':
      return 'Всё штатно — пью кофе и смотрю на метрики.';
    case 'worried':
      return 'Есть дым на горизонте. Слежу за кластерами.';
    case 'panic':
      return 'Горим! Бегу чинить, держитесь!';
  }
}