import type { Alert, Bucket, ClusterSummary, Group, Installation, NotifyLevel, Snapshot } from './types';

export type WallEvent =
  | { type: 'alert.firing'; cluster: string; alert: Alert }
  | { type: 'alert.resolved'; cluster: string; alert: Alert }
  | { type: 'group.opened'; cluster: string; group: Group }
  | { type: 'group.promoted'; cluster: string; group: Group }
  | { type: 'group.resolved'; cluster: string; group: Group }
  | { type: 'cluster.down'; cluster: string }
  | { type: 'cluster.up'; cluster: string };

const severityRank: Record<Bucket, number> = { critical: 0, warning: 1, info: 2 };

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
      if (hasGroups(nextCluster)) {
        for (const group of nextCluster.groups ?? []) {
          events.push({ type: 'group.opened', cluster: nextCluster.name, group });
        }
      } else {
        for (const alert of nextCluster.alerts) {
          events.push({ type: 'alert.firing', cluster: nextCluster.name, alert });
        }
      }
      continue;
    }
    if (prevCluster.reachable && !nextCluster.reachable) {
      events.push({ type: 'cluster.down', cluster: nextCluster.name });
    }
    if (!prevCluster.reachable && nextCluster.reachable) {
      events.push({ type: 'cluster.up', cluster: nextCluster.name });
    }
    if (hasGroups(prevCluster) || hasGroups(nextCluster)) {
      events.push(...diffGroups(nextCluster.name, prevCluster, nextCluster));
      continue;
    }
    const prevFiring = fingerprints(prevCluster.alerts);
    const prevSilenced = fingerprints(prevCluster.silenced);
    const nextFiring = fingerprints(nextCluster.alerts);
    const nextSilenced = fingerprints(nextCluster.silenced);
    for (const alert of nextCluster.alerts) {
      const wasFiring = prevFiring.has(alert.fingerprint);
      const wasSilenced = prevSilenced.has(alert.fingerprint);
      if (!wasFiring && !wasSilenced) {
        events.push({ type: 'alert.firing', cluster: nextCluster.name, alert });
      } else if (wasSilenced) {
        events.push({ type: 'alert.firing', cluster: nextCluster.name, alert });
      }
    }
    for (const alert of prevCluster.alerts) {
      if (!nextFiring.has(alert.fingerprint) && !nextSilenced.has(alert.fingerprint)) {
        events.push({ type: 'alert.resolved', cluster: nextCluster.name, alert });
      }
    }
    for (const alert of prevCluster.silenced ?? []) {
      if (!nextFiring.has(alert.fingerprint) && !nextSilenced.has(alert.fingerprint)) {
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

function hasGroups(cluster: ClusterSummary | undefined): boolean {
  return (cluster?.groups?.length ?? 0) > 0;
}

function diffGroups(cluster: string, prev: ClusterSummary, next: ClusterSummary): WallEvent[] {
  const prevFiring = fingerprints(prev.alerts);
  const nextFiring = fingerprints(next.alerts);
  const nextSilenced = fingerprints(next.silenced);
  const events: WallEvent[] = [];
  for (const group of next.groups ?? []) {
    if (!group.members.some((member) => prevFiring.has(member.fingerprint))) {
      events.push({ type: 'group.opened', cluster, group });
      continue;
    }
    const prevSeverity = overlapSeverity(prev, group.members);
    if (prevSeverity && severityRank[group.severity] < severityRank[prevSeverity]) {
      events.push({ type: 'group.promoted', cluster, group });
    }
  }
  for (const group of prev.groups ?? []) {
    const still = group.members.some((member) => nextFiring.has(member.fingerprint));
    const silenced = group.members.some((member) => nextSilenced.has(member.fingerprint));
    if (!still && !silenced) events.push({ type: 'group.resolved', cluster, group });
  }
  return events;
}

function overlapSeverity(prev: ClusterSummary, members: Alert[]): Bucket | null {
  const want = new Set(members.map((member) => member.fingerprint));
  let best: Bucket | null = null;
  const consider = (severity: Bucket) => {
    if (best === null || severityRank[severity] < severityRank[best]) best = severity;
  };
  if (hasGroups(prev)) {
    for (const group of prev.groups ?? []) {
      if (group.members.some((member) => want.has(member.fingerprint))) consider(group.severity);
    }
    return best;
  }
  for (const alert of prev.alerts) {
    if (want.has(alert.fingerprint)) consider(alert.severity);
  }
  return best;
}

function fingerprints(alerts: { fingerprint: string }[] | undefined): Set<string> {
  return new Set((alerts ?? []).map((alert) => alert.fingerprint));
}

function openedBody(cluster: string, group: Group): string {
  const extra = group.members.length - 1;
  if (extra < 1) return cluster ? `${group.root.alertname} · ${cluster}` : group.root.alertname;
  if (!cluster) return `${group.root.alertname} · ещё ${extra}`;
  return `${group.root.alertname} · ${cluster} · ещё ${extra}`;
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
  const notices: Notice[] = [];
  const grouped = events.some((event) => event.type.startsWith('group.'));
  if (grouped) {
    const opened = events.filter(
      (event): event is Extract<WallEvent, { type: 'group.opened' }> =>
        event.type === 'group.opened' &&
        event.group.root.alertname !== '' &&
        severityAllowed(install.notify, event.group.severity),
    );
    const promoted = events.filter(
      (event): event is Extract<WallEvent, { type: 'group.promoted' }> =>
        event.type === 'group.promoted' &&
        event.group.root.alertname !== '' &&
        severityAllowed(install.notify, event.group.severity),
    );
    if (opened.length === 1) {
      notices.push({ title: install.label, body: openedBody(opened[0].cluster, opened[0].group), tag: `${install.id}:firing` });
    } else if (opened.length > 1) {
      notices.push({ title: install.label, body: `${opened.length} новых групп`, tag: `${install.id}:firing` });
    } else if (promoted.length === 1) {
      const event = promoted[0];
      const cluster = event.cluster ? `${event.group.root.alertname} · ${event.cluster}` : event.group.root.alertname;
      notices.push({
        title: install.label,
        body: `${cluster} · ${event.group.severity}`,
        tag: `${install.id}:firing`,
      });
    } else if (promoted.length > 1) {
      notices.push({ title: install.label, body: `${promoted.length} групп стали тяжелее`, tag: `${install.id}:firing` });
    }
  } else {
    const names: string[] = [];
    let cluster = '';
    for (const event of events) {
      if (event.type !== 'alert.firing' || !event.alert.alertname) continue;
      if (!severityAllowed(install.notify, event.alert.severity)) continue;
      names.push(event.alert.alertname);
      cluster = event.cluster;
    }
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
