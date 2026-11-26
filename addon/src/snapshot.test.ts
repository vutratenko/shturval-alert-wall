import { describe, expect, it } from 'vitest';
import { parseSnapshot, takeSSEEvents } from './snapshot';

const snapshot = {
  generatedAt: '2026-11-26T12:00:00Z',
  mood: 'calm',
  clusters: [],
};

describe('parseSnapshot', () => {
  it('reads a snapshot and a stream frame', () => {
    expect(parseSnapshot(snapshot)?.mood).toBe('calm');
    expect(parseSnapshot({ type: 'snapshot', snapshot })?.clusters).toEqual([]);
    expect(parseSnapshot({ type: 'alert.firing' })).toBeNull();
    expect(parseSnapshot(null)).toBeNull();
  });
});

describe('takeSSEEvents', () => {
  it('splits complete events and keeps a partial chunk', () => {
    const raw = ': keepalive\n\nevent: snapshot\ndata: {"mood":"calm","clusters":[]}\n\nevent: alert';
    const parsed = takeSSEEvents(raw);
    expect(parsed.events).toEqual([
      { event: 'snapshot', data: '{"mood":"calm","clusters":[]}' },
    ]);
    expect(parsed.rest.startsWith('event: alert')).toBe(true);
  });
});
