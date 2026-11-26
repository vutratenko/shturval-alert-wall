import { describe, expect, it } from 'vitest';
import { alertChips, boardCards, filterAlerts, firingDuration, groupDetail, groupReason, memberLine, sortAlerts } from './alerts';
import type { Alert, Group, SilencedAlert } from './types';

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

describe('memberLine', () => {
  it('names the alert, its severity, and how long it has fired', () => {
    const now = new Date('2026-03-20T12:00:00Z').getTime();
    expect(
      memberLine(base({ alertname: 'KubePodNotReady', severity: 'warning', startsAt: '2026-03-20T11:48:00Z' }), now),
    ).toBe('KubePodNotReady · warning · горит 12 мин');
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
    expect(boardCards(alerts, silenced, 'all').map((card) => (card.kind === 'group' ? card.group.id : card.alert.fingerprint))).toEqual(['w', 's']);
    expect(boardCards(alerts, silenced, 'critical')).toEqual([]);
    expect(boardCards(alerts, silenced, 'silence').map((card) => card.kind)).toEqual(['silenced']);
  });

  it('lists groups and keeps silence out of severity filters', () => {
    const root = base({ fingerprint: 'n', severity: 'critical', alertname: 'KubeNodeNotReady' });
    const pod = base({ fingerprint: 'p', severity: 'warning', alertname: 'KubePodNotReady' });
    const group: Group = {
      id: 'g1',
      keyKind: 'node',
      keyValue: 'worker-04',
      severity: 'critical',
      startsAt: root.startsAt,
      root,
      members: [root, pod],
    };
    expect(groupReason(group)).toBe('общий node worker-04');
    expect(groupDetail(group)).toBe('ещё 1 · общий node worker-04');
    expect(groupReason({ ...group, keyKind: 'pod', keyValue: 'orders/checkout' })).toBe('общий pod orders/checkout');
    expect(groupReason({ ...group, keyKind: 'workload', keyValue: 'ml/statefulset/training-worker' })).toBe(
      'общий statefulset ml/training-worker',
    );
    expect(groupReason({ ...group, keyKind: 'alert', keyValue: 'n' })).toBe('');
    const cards = boardCards([], silenced, 'all', [group]);
    expect(cards.map((card) => (card.kind === 'group' ? card.group.id : card.alert.fingerprint))).toEqual(['g1', 's']);
    expect(boardCards([], silenced, 'warning', [group])).toEqual([]);
    expect(boardCards([], silenced, 'critical', [group])[0]?.kind).toBe('group');
  });
});