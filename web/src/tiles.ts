import type { ClusterSummary } from './types';

export type TileTone = 'ok' | 'warn' | 'crit' | 'down';

export function tileTone(c: ClusterSummary): TileTone {
  if (!c.reachable) return 'down';
  if (c.counts.critical > 0) return 'crit';
  if (c.counts.warning > 0) return 'warn';
  return 'ok';
}

export function sortClusters(clusters: ClusterSummary[]): ClusterSummary[] {
  return [...clusters].sort((a, b) => {
    if (a.reachable !== b.reachable) return a.reachable ? -1 : 1;
    const sa = a.counts.critical * 100 + a.counts.warning * 10 + a.counts.info;
    const sb = b.counts.critical * 100 + b.counts.warning * 10 + b.counts.info;
    if (sa !== sb) return sb - sa;
    const ta = a.counts.critical + a.counts.warning + a.counts.info;
    const tb = b.counts.critical + b.counts.warning + b.counts.info;
    if (ta !== tb) return tb - ta;
    return a.name.localeCompare(b.name);
  });
}

export function clusterTotal(c: ClusterSummary): number {
  return c.counts.critical + c.counts.warning + c.counts.info;
}