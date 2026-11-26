import { describe, expect, it } from 'vitest';
import { alertChips, boardCards, filterAlerts, firingDuration, sortAlerts } from './alerts';
import type { Alert, SilencedAlert } from './types';

const base = (overrides: Partial<Alert>): Alert => ({
  fingerprint: 'x',
  alertname: 'Test',
  severity: 'warning',
  startsAt: '2026-03-20T10:00:00Z',
  ...overrides,
});

describe('sortAlerts', () => {
  it('orders by severity then age', () => {
    const alerts = [
      base({ fingerprint: 'i', severity: 'info', startsAt: '2026-03-20T09:00:00Z' }),
      base({ fingerprint: 'c', severity: 'critical', startsAt: '2026-03-20T11:00:00Z' }),
      base({ fingerprint: 'w1', severity: 'warning', startsAt: '2026-03-20T08:00:00Z' }),
      base({ fingerprint: 'w2', severity: 'warning', startsAt: '2026-03-20T12:00:00Z' }),
    ];
    expect(sortAlerts(alerts).map((a) => a.fingerprint)).toEqual(['c', 'w1', 'w2', 'i']);
  });
});

describe('firingDuration', () => {
  it('formats hours', () => {
    const now = new Date('2026-03-20T12:30:00Z').getTime();
    expect(firingDuration('2026-03-20T10:00:00Z', now)).toBe('2 ч 30 мин');
  });
});

describe('alertChips', () => {
  it('picks known label keys', () => {
    const chips = alertChips(
      base({
        namespace: 'orders',
        labels: { pod: 'api-1', node: 'n1', alertname: 'X' },
      }),
    );
    expect(chips).toEqual([
      { key: 'namespace', value: 'orders' },
      { key: 'pod', value: 'api-1' },
      { key: 'node', value: 'n1' },
    ]);
  });
});

describe('filterAlerts', () => {
  it('filters by bucket', () => {
    const alerts = [
      base({ fingerprint: 'c', severity: 'critical' }),
      base({ fingerprint: 'w', severity: 'warning' }),
    ];
    expect(filterAlerts(alerts, 'critical').map((a) => a.fingerprint)).toEqual(['c']);
  });
});

describe('boardCards', () => {
  const silenced: SilencedAlert[] = [
    { ...base({ fingerprint: 's', severity: 'critical' }), silences: [{ id: 'sil-1', endsAt: '2026-03-20T18:00:00Z' }] },
  ];

  it('puts silenced after firing and keeps them out of severity filters', () => {
    const alerts = [base({ fingerprint: 'w', severity: 'warning' })];
    expect(boardCards(alerts, silenced, 'all').map((card) => card.alert.fingerprint)).toEqual(['w', 's']);
    expect(boardCards(alerts, silenced, 'critical')).toEqual([]);
    expect(boardCards(alerts, silenced, 'silence').map((card) => card.kind)).toEqual(['silenced']);
  });
});