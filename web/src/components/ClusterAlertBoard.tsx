import { useEffect, useMemo, useState } from 'react';
import { alertChips, boardCards, firingDuration, severityLabel, type BoardFilter } from '../alerts';
import type { Alert, ClusterSummary, SilenceRef, SilencedAlert } from '../types';

type Filter = BoardFilter;

const durations = [
  { id: '1h', label: '1 ч' },
  { id: '4h', label: '4 ч' },
  { id: '24h', label: '24 ч' },
] as const;

type Duration = (typeof durations)[number]['id'];

type Props = {
  cluster: ClusterSummary | null;
  silenceEnabled: boolean;
  onClose: () => void;
};

export function ClusterAlertBoard({ cluster, silenceEnabled, onClose }: Props) {
  const [filter, setFilter] = useState<Filter>('all');
  const [draftFp, setDraftFp] = useState<string | null>(null);
  const [duration, setDuration] = useState<Duration>('1h');
  const [comment, setComment] = useState('');
  const [confirmFp, setConfirmFp] = useState<string | null>(null);
  const [busy, setBusy] = useState<{ fp: string; action: 'silence' | 'unsilence' } | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!cluster) return;
    setFilter('all');
    setDraftFp(null);
    setConfirmFp(null);
    setBusy(null);
    setError(null);
    setComment('');
  }, [cluster?.name]);

  useEffect(() => {
    if (!busy || !cluster) return;
    const firing = cluster.alerts.some((alert) => alert.fingerprint === busy.fp);
    const silenced = (cluster.silenced ?? []).some((alert) => alert.fingerprint === busy.fp);
    if (busy.action === 'silence' && !firing) setBusy(null);
    if (busy.action === 'unsilence' && !silenced) setBusy(null);
  }, [busy, cluster]);

  useEffect(() => {
    if (!cluster) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [cluster, onClose]);

  useEffect(() => {
    if (!cluster) return;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = previousOverflow;
    };
  }, [cluster]);

  const cards = useMemo(
    () => (cluster ? boardCards(cluster.alerts, cluster.silenced, filter) : []),
    [cluster, filter],
  );

  if (!cluster) return null;

  const { counts } = cluster;
  const total = counts.critical + counts.warning + counts.info;
  const silencedCount = cluster.silenced?.length ?? 0;
  const source = cluster.source || cluster.name;

  async function silence(alert: Alert) {
    const text = comment.trim();
    if (!text) {
      setError('Нужен комментарий');
      return;
    }
    setError(null);
    setBusy({ fp: alert.fingerprint, action: 'silence' });
    const res = await fetch('/api/silences', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source, fingerprint: alert.fingerprint, duration, comment: text }),
    });
    if (!res.ok) {
      setBusy(null);
      setError('Не удалось заглушить');
      return;
    }
    setDraftFp(null);
    setComment('');
  }

  async function unsilence(alert: SilencedAlert) {
    setError(null);
    setBusy({ fp: alert.fingerprint, action: 'unsilence' });
    for (const item of alert.silences) {
      const res = await fetch(`/api/silences/${encodeURIComponent(item.id)}?source=${encodeURIComponent(source)}`, {
        method: 'DELETE',
        credentials: 'same-origin',
      });
      if (!res.ok) {
        setBusy(null);
        setError('Не удалось снять silence');
        return;
      }
    }
    setConfirmFp(null);
  }

  return (
    <div className="board-overlay" role="dialog" aria-modal="true" aria-label={`Алерты ${cluster.name}`} onClick={onClose}>
      <div className="board-inner" onClick={(e) => e.stopPropagation()}>
        <header className="board-top">
          <div>
            <h2>{cluster.name}</h2>
            <p className="sub">
              {cluster.reachable ? 'Активные firing-алерты' : 'Кластер недоступен — показан последний снимок'}
            </p>
          </div>
          <button type="button" className="btn-close" onClick={onClose}>
            Закрыть
          </button>
        </header>
        <div className="board-filters">
          <FilterChip active={filter === 'all'} onClick={() => setFilter('all')}>
            все {total}
          </FilterChip>
          <FilterChip active={filter === 'critical'} className="crit" onClick={() => setFilter('critical')}>
            critical {counts.critical}
          </FilterChip>
          <FilterChip active={filter === 'warning'} className="warn" onClick={() => setFilter('warning')}>
            warning {counts.warning}
          </FilterChip>
          <FilterChip active={filter === 'info'} onClick={() => setFilter('info')}>
            info {counts.info}
          </FilterChip>
          <FilterChip active={filter === 'silence'} onClick={() => setFilter('silence')}>
            silence {silencedCount}
          </FilterChip>
        </div>
        {cards.length === 0 ? (
          <p style={{ color: 'var(--muted)', fontSize: 18 }}>Нет алертов в этой группе</p>
        ) : (
          <div className="alert-grid">
            {cards.map((card) =>
              card.kind === 'firing' ? (
                <article key={card.alert.fingerprint} className={`alert-card sev-${card.alert.severity}`}>
                  <CardBody alert={card.alert} />
                  {silenceEnabled ? (
                    <SilenceDraft
                      alert={card.alert}
                      open={draftFp === card.alert.fingerprint}
                      duration={duration}
                      comment={comment}
                      busy={busy?.fp === card.alert.fingerprint}
                      error={draftFp === card.alert.fingerprint ? error : null}
                      onOpen={() => {
                        setDraftFp(card.alert.fingerprint);
                        setConfirmFp(null);
                        setError(null);
                      }}
                      onDuration={setDuration}
                      onComment={setComment}
                      onSubmit={() => void silence(card.alert)}
                    />
                  ) : null}
                </article>
              ) : (
                <article key={card.alert.fingerprint} className={`alert-card silenced sev-${card.alert.severity}`}>
                  <CardBody alert={card.alert} />
                  <div className="silence-note">
                    {card.alert.silences.map((item) => (
                      <p key={item.id}>
                        silence до {formatSilenceUntil(item.endsAt)}
                        {item.createdBy ? ` · ${item.createdBy}` : ''}
                        {item.comment ? ` — ${item.comment}` : ''}
                      </p>
                    ))}
                  </div>
                  {silenceEnabled ? (
                    <Unsilence
                      alert={card.alert}
                      open={confirmFp === card.alert.fingerprint}
                      busy={busy?.fp === card.alert.fingerprint}
                      error={confirmFp === card.alert.fingerprint ? error : null}
                      onOpen={() => {
                        setConfirmFp(card.alert.fingerprint);
                        setDraftFp(null);
                        setError(null);
                      }}
                      onCancel={() => setConfirmFp(null)}
                      onSubmit={() => void unsilence(card.alert)}
                    />
                  ) : null}
                </article>
              ),
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function CardBody({ alert }: { alert: Alert }) {
  return (
    <>
      <div className="meta">
        <span className="sev">
          <span className="dot" aria-hidden />
          <span className="sev-label">{severityLabel(alert.severity)}</span>
        </span>
        <span className="age mono">горит {firingDuration(alert.startsAt)}</span>
      </div>
      <h3 className="title">{alert.alertname}</h3>
      {alert.summary ? <p className="summary">{alert.summary}</p> : null}
      {alert.description ? <p className="desc">{alert.description}</p> : null}
      <div className="chips">
        {alertChips(alert).map((c) => (
          <span key={`${c.key}-${c.value}`}>
            {c.key}={c.value}
          </span>
        ))}
      </div>
      <div className="links">
        {alert.runbookUrl ? (
          <a href={alert.runbookUrl} target="_blank" rel="noreferrer">
            Runbook
          </a>
        ) : null}
        {alert.generatorUrl ? (
          <a href={alert.generatorUrl} target="_blank" rel="noreferrer">
            Источник
          </a>
        ) : null}
      </div>
    </>
  );
}

function SilenceDraft({
  alert,
  open,
  duration,
  comment,
  busy,
  error,
  onOpen,
  onDuration,
  onComment,
  onSubmit,
}: {
  alert: Alert;
  open: boolean;
  duration: Duration;
  comment: string;
  busy: boolean;
  error: string | null;
  onOpen: () => void;
  onDuration: (duration: Duration) => void;
  onComment: (comment: string) => void;
  onSubmit: () => void;
}) {
  if (!open) {
    return (
      <div className="row-actions">
        <button type="button" className="btn-quiet" onClick={onOpen} disabled={busy}>
          {busy ? 'Заглушается…' : 'Заглушить'}
        </button>
      </div>
    );
  }
  const labels = Object.entries(alert.labels ?? {}).sort(([a], [b]) => a.localeCompare(b));
  return (
    <form
      className="silence-form"
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit();
      }}
    >
      <div className="row-actions">
        {durations.map((item) => (
          <button
            key={item.id}
            type="button"
            className={`chip${duration === item.id ? ' on' : ''}`}
            onClick={() => onDuration(item.id)}
          >
            {item.label}
          </button>
        ))}
      </div>
      <label>
        Комментарий
        <textarea value={comment} onChange={(e) => onComment(e.target.value)} required maxLength={1024} />
      </label>
      {labels.length > 0 ? (
        <div className="chips">
          {labels.map(([key, value]) => (
            <span key={`${key}-${value}`}>
              {key}={value}
            </span>
          ))}
        </div>
      ) : null}
      {error ? <p className="form-error">{error}</p> : null}
      <div className="row-actions">
        <button type="submit" className="btn-quiet" disabled={busy || comment.trim() === ''}>
          {busy ? 'Заглушается…' : 'Заглушить'}
        </button>
      </div>
    </form>
  );
}

function Unsilence({
  alert,
  open,
  busy,
  error,
  onOpen,
  onCancel,
  onSubmit,
}: {
  alert: SilencedAlert;
  open: boolean;
  busy: boolean;
  error: string | null;
  onOpen: () => void;
  onCancel: () => void;
  onSubmit: () => void;
}) {
  if (!open) {
    return (
      <div className="row-actions">
        <button type="button" className="btn-quiet" onClick={onOpen} disabled={busy}>
          {busy ? 'Снимается…' : 'Снять silence'}
        </button>
      </div>
    );
  }
  return (
    <div className="silence-confirm">
      <p>Снимет silence целиком. Если он шире этого алерта, зазвучат и остальные совпадения.</p>
      {alert.silences.map((item) => (
        <MatcherList key={item.id} silence={item} alert={alert} />
      ))}
      {error ? <p className="form-error">{error}</p> : null}
      <div className="row-actions">
        <button type="button" className="btn-quiet" onClick={onSubmit} disabled={busy}>
          {busy ? 'Снимается…' : 'Снять silence'}
        </button>
        <button type="button" className="btn-quiet" onClick={onCancel} disabled={busy}>
          Отмена
        </button>
      </div>
    </div>
  );
}

function MatcherList({ silence, alert }: { silence: SilenceRef; alert: Alert }) {
  const matchers = silence.matchers ?? [];
  const narrower = matchers.length > 0 && matchers.length < Object.keys(alert.labels ?? {}).length;
  return (
    <div>
      <div className="chips">
        {matchers.map((matcher) => (
          <span key={`${silence.id}-${matcher.name}-${matcher.value}`}>
            {matcher.name}={matcher.value}
          </span>
        ))}
      </div>
      {narrower ? <p className="silence-note">Этот silence шире карточки.</p> : null}
    </div>
  );
}

function formatSilenceUntil(iso: string): string {
  const t = new Date(iso);
  if (Number.isNaN(t.getTime())) return iso;
  const time = t.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  if (t.toDateString() === new Date().toDateString()) return time;
  const day = t.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' });
  return `${day} ${time}`;
}

function FilterChip({
  children,
  active,
  className,
  onClick,
}: {
  children: React.ReactNode;
  active: boolean;
  className?: string;
  onClick: () => void;
}) {
  return (
    <button type="button" className={`chip${active ? ` on ${className ?? ''}` : ''}`} onClick={onClick}>
      {children}
    </button>
  );
}
