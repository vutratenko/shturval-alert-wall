import type { Alert, Installation, NotifyLevel, Snapshot } from './types';

export type WallEvent =
  | { type: 'alert.firing'; cluster: string; alert: Alert }
  | { type: 'alert.resolved'; cluster: string; alert: Alert }
  | { type: 'cluster.down'; cluster: string }
  | { type: 'cluster.up'; cluster: string };

export type Notice = { title: string; body: string; tag: string };

export function diffSnapshots(prev: Snapshot | null, next: Snapshot): WallEvent[] {
  if (!prev) return [];
  const events: WallEvent[] = [];
  const prevByName = new Map(prev.clusters.map((cluster) => [cluster.name, cluster]));
  const nextByName = new Map(next.clusters.map((cluster) => [cluster.name, cluster]));

  for (const nextCluster of next.clusters) {
    const prevCluster = prevByName.get(nextCluster.name);
    if (!prevCluster) {
      if (!nextCluster.reachable) events.push({ type: 'cluster.down', cluster: nextCluster.name });
      for (const alert of nextCluster.alerts) {
        events.push({ type: 'alert.firing', cluster: nextCluster.name, alert });
      }
      continue;
    }
    if (prevCluster.reachable && !nextCluster.reachable) {
      events.push({ type: 'cluster.down', cluster: nextCluster.name });
    }
    if (!prevCluster.reachable && nextCluster.reachable) {
      events.push({ type: 'cluster.up', cluster: nextCluster.name });
    }
    const prevFingerprints = new Set(prevCluster.alerts.map((alert) => alert.fingerprint));
    const nextFingerprints = new Set(nextCluster.alerts.map((alert) => alert.fingerprint));
    for (const alert of nextCluster.alerts) {
      if (!prevFingerprints.has(alert.fingerprint)) {
        events.push({ type: 'alert.firing', cluster: nextCluster.name, alert });
      }
    }
    for (const alert of prevCluster.alerts) {
      if (!nextFingerprints.has(alert.fingerprint)) {
        events.push({ type: 'alert.resolved', cluster: nextCluster.name, alert });
      }
    }
  }

  for (const prevCluster of prev.clusters) {
    if (!nextByName.has(prevCluster.name) && prevCluster.reachable) {
      events.push({ type: 'cluster.down', cluster: prevCluster.name });
    }
  }
  return events;
}

function severityAllowed(level: NotifyLevel, severity: Alert['severity']): boolean {
  if (level === 'off') return false;
  if (level === 'all') return true;
  if (level === 'critical') return severity === 'critical';
  return severity === 'critical' || severity === 'warning';
}

export function noticesFor(
  install: Pick<Installation, 'id' | 'label' | 'notify' | 'notifyClusterDown'>,
  events: WallEvent[],
): Notice[] {
  const names: string[] = [];
  let cluster = '';
  for (const event of events) {
    if (event.type !== 'alert.firing' || !event.alert.alertname) continue;
    if (!severityAllowed(install.notify, event.alert.severity)) continue;
    names.push(event.alert.alertname);
    cluster = event.cluster;
  }

  const notices: Notice[] = [];
  if (names.length === 1) {
    notices.push({
      title: install.label,
      body: cluster ? `${names[0]} · ${cluster}` : names[0],
      tag: `${install.id}:firing`,
    });
  } else if (names.length > 1) {
    notices.push({
      title: install.label,
      body: `${names.length} новых алертов`,
      tag: `${install.id}:firing`,
    });
  }

  if (install.notifyClusterDown) {
    for (const event of events) {
      if (event.type !== 'cluster.down') continue;
      notices.push({
        title: install.label,
        body: `${event.cluster} недоступен`,
        tag: `${install.id}:down:${event.cluster}`,
      });
    }
  }
  return notices;
}
