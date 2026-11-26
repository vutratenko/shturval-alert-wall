import { describe, expect, it } from 'vitest';
import { mergeView } from './mergeView';
import type { InstallView, Snapshot } from './types';

function snap(critical: number): Snapshot {
  return {
    generatedAt: '2026-11-26T12:00:00Z',
    mood: 'panic',
    clusters: [
      {
        name: 'a',
        reachable: true,
        counts: { critical, warning: 0, info: 0 },
        alerts: [],
      },
    ],
  };
}

const live = (critical: number): InstallView => ({ status: 'ok', snapshot: snap(critical) });

describe('mergeView', () => {
  it('replaces the previous snapshot when the poll succeeds', () => {
    expect(mergeView({ status: 'ok', snapshot: snap(6) }, live(1), live(2)).snapshot?.clusters[0]?.counts.critical).toBe(6);
  });

  it('marks login even if a snapshot was known', () => {
    expect(mergeView({ status: 'needs-login', snapshot: null }, live(6), live(6))).toEqual({
      status: 'needs-login',
      snapshot: null,
    });
  });

  it('keeps a snapshot written while the poll was in flight', () => {
    expect(mergeView({ status: 'offline', snapshot: null }, undefined, live(6))).toEqual(live(6));
  });

  it('keeps the snapshot from the start of the poll when nothing newer arrived', () => {
    expect(mergeView({ status: 'offline', snapshot: null }, live(4), { status: 'offline', snapshot: null })).toEqual(live(4));
  });

  it('stays offline only when no live snapshot exists', () => {
    expect(mergeView({ status: 'offline', snapshot: null }, undefined, undefined)).toEqual({
      status: 'offline',
      snapshot: null,
    });
  });
});
