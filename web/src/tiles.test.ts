import { describe, expect, it } from 'vitest';
import { sortClusters, tileTone } from './tiles';
import type { ClusterSummary } from './types';

describe('tileTone', () => {
  it('marks critical', () => {
    expect(tileTone({ name: 'a', reachable: true, counts: { critical: 1, warning: 0, info: 0 }, alerts: [] })).toBe('crit');
  });
  it('marks down', () => {
    expect(tileTone({ name: 'a', reachable: false, counts: { critical: 0, warning: 0, info: 0 }, alerts: [] })).toBe('down');
  });
});

describe('sortClusters', () => {
  it('puts worse first', () => {
    const a: ClusterSummary = { name: 'a', reachable: true, counts: { critical: 0, warning: 0, info: 1 }, alerts: [] };
    const b: ClusterSummary = { name: 'b', reachable: true, counts: { critical: 1, warning: 0, info: 0 }, alerts: [] };
    expect(sortClusters([a, b])[0].name).toBe('b');
  });
});