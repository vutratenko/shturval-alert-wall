import type { Mood } from './types';

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
