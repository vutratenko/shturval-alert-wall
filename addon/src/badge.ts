import type { Installation, InstallView, Settings, Snapshot } from './types';

export const BADGE_COLORS = {
  calm: '#0f766e',
  worried: '#b86e00',
  panic: '#c62828',
  attention: '#5c6570',
} as const;

export type Badge = { text: string; color: string };

const MOOD_RANK = { panic: 3, worried: 2, calm: 1 } as const;

export function criticalCount(snapshot: Snapshot): number {
  return snapshot.clusters.reduce((sum, cluster) => sum + (cluster.counts?.critical ?? 0), 0);
}

export function pickInstallation(settings: Settings, views: Record<string, InstallView>): Installation | null {
  const enabled = settings.installations.filter((install) => install.enabled);
  if (settings.activeId) {
    const active = enabled.find((install) => install.id === settings.activeId);
    if (active) return active;
  }
  const ranked = enabled
    .filter((install) => views[install.id]?.status === 'ok' && views[install.id]?.snapshot)
    .sort((a, b) => {
      const left = views[a.id].snapshot!;
      const right = views[b.id].snapshot!;
      const byMood = MOOD_RANK[right.mood] - MOOD_RANK[left.mood];
      if (byMood !== 0) return byMood;
      return criticalCount(right) - criticalCount(left);
    });
  if (ranked[0]) return ranked[0];
  return enabled.find((install) => views[install.id]?.status === 'needs-login') ?? enabled[0] ?? null;
}

function badgeText(critical: number): string {
  if (critical <= 0) return '';
  if (critical > 99) return '99+';
  return String(critical);
}

export function badgeFor(settings: Settings, views: Record<string, InstallView>): Badge {
  const install = pickInstallation(settings, views);
  if (!install) return { text: '', color: BADGE_COLORS.attention };
  const view = views[install.id];
  if (!view || view.status === 'offline') return { text: '?', color: BADGE_COLORS.attention };
  if (view.status === 'needs-login') return { text: '!', color: BADGE_COLORS.attention };
  const snapshot = view.snapshot;
  if (!snapshot) return { text: '?', color: BADGE_COLORS.attention };
  return { text: badgeText(criticalCount(snapshot)), color: BADGE_COLORS[snapshot.mood] };
}
