import { describe, expect, it } from 'vitest';
import { diffSnapshots, noticesFor } from './diff';
import type { Alert, Installation, Snapshot } from './types';

function alert(partial: Partial<Alert> & Pick<Alert, 'fingerprint' | 'alertname'>): Alert {
  return { severity: 'critical', ...partial };
}

function snap(clusters: Snapshot['clusters'], mood: Snapshot['mood'] = 'calm'): Snapshot {
  return { generatedAt: '2026-11-26T12:00:00Z', mood, clusters };
}

const install: Installation = {
  id: 'prod',
  label: 'prod',
  baseUrl: 'https://alerts.example.com',
  enabled: true,
  notify: 'critical',
  notifyClusterDown: true,
};

describe('diffSnapshots', () => {
  const base = snap([
    {
      name: 'a',
      reachable: true,
      counts: { critical: 1, warning: 0, info: 0 },
      alerts: [alert({ fingerprint: '1', alertname: 'DiskFull' })],
    },
  ]);

  it('remembers the first snapshot without events', () => {
    expect(diffSnapshots(null, base)).toEqual([]);
  });

  it('reports a new firing alert and a resolved one', () => {
    const next = snap([
      {
        name: 'a',
        reachable: true,
        counts: { critical: 1, warning: 0, info: 0 },
        alerts: [alert({ fingerprint: '2', alertname: 'CPU' })],
      },
    ]);
    expect(diffSnapshots(base, next)).toEqual([
      { type: 'alert.firing', cluster: 'a', alert: alert({ fingerprint: '2', alertname: 'CPU' }) },
      {
        type: 'alert.resolved',
        cluster: 'a',
        alert: alert({ fingerprint: '1', alertname: 'DiskFull' }),
      },
    ]);
  });

  it('keeps a silenced alert off the badge and reports it again when the silence ends', () => {
    const silenced = snap([
      {
        name: 'a',
        reachable: true,
        counts: { critical: 0, warning: 0, info: 0 },
        alerts: [],
        silenced: [alert({ fingerprint: '1', alertname: 'DiskFull' })],
      },
    ]);
    expect(diffSnapshots(base, silenced)).toEqual([]);
    expect(diffSnapshots(silenced, base)).toEqual([
      { type: 'alert.firing', cluster: 'a', alert: alert({ fingerprint: '1', alertname: 'DiskFull' }) },
    ]);
  });

  it('reports cluster reachability and a cluster that disappeared', () => {
    const down = snap([
      { name: 'a', reachable: false, counts: { critical: 0, warning: 0, info: 0 }, alerts: [] },
    ]);
    expect(diffSnapshots(base, down).map((event) => event.type)).toEqual(['cluster.down', 'alert.resolved']);

    const up = snap([
      { name: 'a', reachable: true, counts: { critical: 0, warning: 0, info: 0 }, alerts: [] },
    ]);
    expect(diffSnapshots(down, up)).toEqual([{ type: 'cluster.up', cluster: 'a' }]);

    expect(diffSnapshots(base, snap([]))).toEqual([{ type: 'cluster.down', cluster: 'a' }]);
    expect(diffSnapshots(down, snap([]))).toEqual([]);
  });

  it('treats a new unreachable cluster as down and still reports its alerts', () => {
    const next = snap([
      ...base.clusters,
      {
        name: 'b',
        reachable: false,
        counts: { critical: 1, warning: 0, info: 0 },
        alerts: [alert({ fingerprint: '9', alertname: 'NodeDown' })],
      },
    ]);
    expect(diffSnapshots(base, next)).toEqual([
      { type: 'cluster.down', cluster: 'b' },
      { type: 'alert.firing', cluster: 'b', alert: alert({ fingerprint: '9', alertname: 'NodeDown' }) },
    ]);
  });
});

describe('noticesFor', () => {
  const firing = (name: string, severity: Alert['severity'], cluster = 'a'): ReturnType<typeof diffSnapshots>[number] => ({
    type: 'alert.firing',
    cluster,
    alert: alert({ fingerprint: name, alertname: name, severity }),
  });

  it('uses the installation label and one alert name', () => {
    expect(noticesFor(install, [firing('DiskFull', 'critical')])).toEqual([
      { title: 'prod', body: 'DiskFull · a', tag: 'prod:firing' },
    ]);
  });

  it('coalesces a burst and skips empty names', () => {
    const events = [
      firing('A', 'critical'),
      { type: 'alert.firing' as const, cluster: 'b', alert: alert({ fingerprint: 'x', alertname: '', severity: 'critical' }) },
      firing('B', 'critical', 'b'),
    ];
    expect(noticesFor(install, events)).toEqual([
      { title: 'prod', body: '2 новых алертов', tag: 'prod:firing' },
    ]);
  });

  it('filters by notify level and keeps cluster.down independent', () => {
    const events = [firing('DiskFull', 'critical'), firing('Slow', 'warning'), { type: 'cluster.down' as const, cluster: 'a' }];
    expect(noticesFor({ ...install, notify: 'critical' }, events).map((notice) => notice.body)).toEqual([
      'DiskFull · a',
      'a недоступен',
    ]);
    expect(noticesFor({ ...install, notify: 'warning' }, events)[0]?.body).toBe('2 новых алертов');
    expect(noticesFor({ ...install, notify: 'off' }, events)).toEqual([
      { title: 'prod', body: 'a недоступен', tag: 'prod:down:a' },
    ]);
    expect(noticesFor({ ...install, notify: 'off', notifyClusterDown: false }, events)).toEqual([]);
    expect(noticesFor({ ...install, notify: 'all' }, [firing('Hint', 'info')])[0]?.body).toBe('Hint · a');
  });
});
