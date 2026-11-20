import { describe, expect, it } from 'vitest';
import { nextMood } from './moodHysteresis';

describe('nextMood', () => {
  it('escalates immediately', () => {
    const s = nextMood({ displayed: 'calm' }, 'panic', 0);
    expect(s.displayed).toBe('panic');
  });
  it('delays de-escalation', () => {
    let s = nextMood({ displayed: 'panic' }, 'calm', 0);
    expect(s.displayed).toBe('panic');
    s = nextMood(s, 'calm', 121_000);
    expect(s.displayed).toBe('calm');
  });
});