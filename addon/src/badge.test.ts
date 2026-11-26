import { describe, expect, it } from 'vitest';
import { BADGE_COLORS, badgeFor } from './badge';
import type { InstallView, Settings, Snapshot } from './types';

function snap(mood: Snapshot['mood'], critical: number): Snapshot {
  return {
    generatedAt: '2026-11-26T12:00:00Z',
    mood,
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

function view(snapshot: Snapshot | null, status: InstallView['status'] = 'ok'): InstallView {
  return { status, snapshot };
}

function settings(activeId: string | null, enabled = true): Settings {
  return {
    activeId,
    installations: [
      {
        id: 'calm',
        label: 'calm',
        baseUrl: 'https://calm.example',
        enabled,
        notify: 'critical',
        notifyClusterDown: true,
      },
      {
        id: 'panic',
        label: 'panic',
        baseUrl: 'https://panic.example',
        enabled: true,
        notify: 'critical',
        notifyClusterDown: true,
      },
    ],
  };
}

describe('badgeFor', () => {
  it('shows the active wall critical count and mood color', () => {
    const views = {
      calm: view(snap('calm', 0)),
      panic: view(snap('panic', 3)),
    };
    expect(badgeFor(settings('panic'), views)).toEqual({ text: '3', color: BADGE_COLORS.panic });
    expect(badgeFor(settings('calm'), views)).toEqual({ text: '', color: BADGE_COLORS.calm });
  });

  it('caps the count and marks login and offline', () => {
    expect(badgeFor(settings('panic'), { panic: view(snap('panic', 120)) }).text).toBe('99+');
    expect(badgeFor(settings('panic'), { panic: view(null, 'needs-login') })).toEqual({
      text: '!',
      color: BADGE_COLORS.attention,
    });
    expect(badgeFor(settings('panic'), { panic: view(null, 'offline') }).text).toBe('?');
  });

  it('picks the worst enabled wall when none is active', () => {
    const views = {
      calm: view(snap('worried', 4)),
      panic: view(snap('panic', 1)),
    };
    expect(badgeFor(settings(null), views).text).toBe('1');
    expect(badgeFor(settings(null), { calm: view(snap('worried', 4)), panic: view(snap('worried', 1)) }).text).toBe('4');
  });

  it('ignores a disabled active wall and falls back to login when nobody is live', () => {
    expect(badgeFor(settings('calm', false), { panic: view(snap('calm', 2)) }).text).toBe('2');
    expect(
      badgeFor(settings(null), { calm: view(null, 'needs-login'), panic: view(null, 'offline') }).text,
    ).toBe('!');
    expect(badgeFor({ activeId: null, installations: [] }, {})).toEqual({
      text: '',
      color: BADGE_COLORS.attention,
    });
  });
});
