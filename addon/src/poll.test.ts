import { describe, expect, it } from 'vitest';
import { fetchSnapshot, interpretPoll } from './poll';
import type { Installation, Snapshot } from './types';

const install: Installation = {
  id: 'prod',
  label: 'prod',
  baseUrl: 'https://alerts.example.com',
  enabled: true,
  notify: 'critical',
  notifyClusterDown: false,
};

function snap(fingerprint: string): Snapshot {
  return {
    generatedAt: '2026-11-26T12:00:00Z',
    mood: 'panic',
    clusters: [
      {
        name: 'a',
        reachable: true,
        counts: { critical: 1, warning: 0, info: 0 },
        alerts: [{ fingerprint, alertname: 'DiskFull', severity: 'critical' }],
      },
    ],
  };
}

describe('interpretPoll', () => {
  it('baselines the first snapshot and notifies only the next new alert', () => {
    const first = interpretPoll(install, null, { status: 200, body: snap('1') });
    expect(first.notices).toEqual([]);
    expect(first.remembered?.clusters[0]?.alerts[0]?.fingerprint).toBe('1');
    const second = interpretPoll(install, first.remembered, { status: 200, body: snap('2') });
    expect(second.notices).toEqual([{ title: 'prod', body: 'DiskFull · a', tag: 'prod:firing' }]);
  });

  it('does not notify on login, redirect, or a network error', () => {
    const prev = snap('1');
    for (const response of [
      { status: 401, body: null },
      { status: 302, body: null, redirected: false },
      { error: true as const },
    ]) {
      const result = interpretPoll(install, prev, response);
      expect(result.notices).toEqual([]);
      expect(result.remembered).toBe(prev);
    }
    expect(interpretPoll(install, prev, { status: 401, body: null }).status).toBe('needs-login');
    expect(interpretPoll(install, prev, { status: 302, body: null }).status).toBe('needs-login');
    expect(interpretPoll(install, prev, { error: true }).status).toBe('offline');
    expect(interpretPoll(install, prev, { status: 200, body: { no: 'snapshot' } }).status).toBe('offline');
  });
});

describe('fetchSnapshot', () => {
  it('reads json and treats an opaque redirect as login', async () => {
    const ok = await fetchSnapshot('https://alerts.example.com', async () => new Response(JSON.stringify(snap('1')), { status: 200 }));
    expect(ok).toMatchObject({ status: 200 });
    const redirected = await fetchSnapshot('https://alerts.example.com', async () => ({
      status: 0,
      type: 'opaqueredirect',
      text: async () => '',
    }) as Response);
    expect(redirected).toMatchObject({ redirected: true, status: 401 });
    const failed = await fetchSnapshot('https://alerts.example.com', async () => {
      throw new Error('down');
    });
    expect(failed).toEqual({ error: true });
  });
});
