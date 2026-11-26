import type { Alert, Bucket, Counts, SilencedAlert, Snapshot } from './types';

const rank: Record<Bucket, number> = { critical: 0, warning: 1, info: 2 };

const chipKeys = ['namespace', 'pod', 'node', 'job', 'instance', 'container'] as const;

export function sortAlerts(alerts: Alert[]): Alert[] {
  return [...alerts].sort((a, b) => {
    const dr = rank[a.severity] - rank[b.severity];
    if (dr !== 0) return dr;
    return new Date(a.startsAt).getTime() - new Date(b.startsAt).getTime();
  });
}

export function firingDuration(startsAt: string, now = Date.now()): string {
  const ms = Math.max(0, now - new Date(startsAt).getTime());
  const sec = Math.floor(ms / 1000);
  if (sec < 60) return `${sec} сек`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min} мин`;
  const h = Math.floor(min / 60);
  const rm = min % 60;
  if (h < 24) return rm > 0 ? `${h} ч ${rm} мин` : `${h} ч`;
  const d = Math.floor(h / 24);
  const rh = h % 24;
  return rh > 0 ? `${d} д ${rh} ч` : `${d} д`;
}

export function alertChips(alert: Alert): { key: string; value: string }[] {
  const labels = alert.labels ?? {};
  const out: { key: string; value: string }[] = [];
  for (const key of chipKeys) {
    const fromLabel = labels[key];
    const direct = key === 'namespace' ? alert.namespace : undefined;
    const value = fromLabel ?? direct;
    if (value) out.push({ key, value });
  }
  return out;
}

export function snapshotTotals(snapshot: Snapshot | null): Counts & { clustersFiring: number } {
  const empty = { critical: 0, warning: 0, info: 0, clustersFiring: 0 };
  if (!snapshot) return empty;
  let critical = 0;
  let warning = 0;
  let info = 0;
  let clustersFiring = 0;
  for (const c of snapshot.clusters) {
    critical += c.counts.critical;
    warning += c.counts.warning;
    info += c.counts.info;
    if (c.counts.critical + c.counts.warning + c.counts.info > 0) clustersFiring++;
  }
  return { critical, warning, info, clustersFiring };
}

export function filterAlerts(alerts: Alert[], filter: Bucket | 'all'): Alert[] {
  const sorted = sortAlerts(alerts);
  if (filter === 'all') return sorted;
  return sorted.filter((a) => a.severity === filter);
}

export type BoardFilter = Bucket | 'all' | 'silence';

export type BoardCard =
  | { kind: 'firing'; alert: Alert }
  | { kind: 'silenced'; alert: SilencedAlert };

export function boardCards(alerts: Alert[], silenced: SilencedAlert[] | undefined, filter: BoardFilter): BoardCard[] {
  const quiet = silenced ?? [];
  if (filter === 'silence') return quiet.map((alert) => ({ kind: 'silenced', alert }));
  const firing = filterAlerts(alerts, filter === 'all' ? 'all' : filter).map(
    (alert) => ({ kind: 'firing', alert }) as BoardCard,
  );
  if (filter !== 'all') return firing;
  return [...firing, ...quiet.map((alert) => ({ kind: 'silenced', alert }) as BoardCard)];
}

export function severityLabel(sev: Bucket): string {
  switch (sev) {
    case 'critical':
      return 'critical';
    case 'warning':
      return 'warning';
    default:
      return 'info';
  }
}