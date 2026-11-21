import { describe, expect, it } from 'vitest';
import { snapshotFromFrame } from './snapshotFrame';
import type { Snapshot } from './types';

const snap = (generatedAt: string): Snapshot => ({
  generatedAt,
  mood: 'calm',
  clusters: [],
});

describe('snapshotFromFrame', () => {
  it('reads the first raw snapshot frame', () => {
    const frame = snap('2026-03-20T10:00:00Z');
    expect(snapshotFromFrame(frame)).toEqual(frame);
  });

  it('reads a later event wrapped in snapshot', () => {
    const inner = snap('2026-03-20T10:00:01Z');
    expect(
      snapshotFromFrame({
        type: 'snapshot',
        at: '2026-03-20T10:00:01Z',
        snapshot: inner,
      }),
    ).toEqual(inner);
  });

  it('ignores frames without a snapshot', () => {
    expect(snapshotFromFrame({ type: 'alert.firing' })).toBeNull();
    expect(snapshotFromFrame(null)).toBeNull();
  });
});
