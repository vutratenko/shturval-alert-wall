import type { StreamEvent } from '../types';

type Props = { events: StreamEvent[] };

function label(ev: StreamEvent): string {
  switch (ev.type) {
    case 'alert.firing':
      return ev.alert?.alertname ?? 'алерт';
    case 'alert.resolved':
      return `${ev.alert?.alertname ?? 'алерт'} погас`;
    case 'cluster.down':
      return 'кластер недоступен';
    case 'cluster.up':
      return 'кластер снова в сети';
    default:
      return ev.type;
  }
}

function rowClass(ev: StreamEvent): string {
  if (ev.type === 'alert.firing') return 'firing';
  if (ev.type === 'alert.resolved') return 'resolved';
  return '';
}

export function EventFeed({ events }: Props) {
  return (
    <div className="card tint event-feed">
      <h3>Лента</h3>
      <div className="event-list">
        {events.length === 0 ? (
          <span style={{ color: 'var(--muted)' }}>Пока тихо</span>
        ) : (
          events.map((ev, i) => (
            <div key={`${ev.type}-${ev.at ?? i}-${ev.alert?.fingerprint ?? ev.cluster}`} className={`event-row ${rowClass(ev)}`}>
              <span className="mono">{timeOf(ev)}</span>
              <span>
                {ev.cluster ? <span className="mono">{ev.cluster} · </span> : null}
                <b>{label(ev)}</b>
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function timeOf(ev: StreamEvent): string {
  if (!ev.at) return '—';
  try {
    return new Date(ev.at).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  } catch {
    return '—';
  }
}