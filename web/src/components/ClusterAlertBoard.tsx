import { useEffect, useMemo, useState } from 'react';
import { alertChips, filterAlerts, firingDuration, severityLabel } from '../alerts';
import type { Bucket, ClusterSummary } from '../types';

type Filter = Bucket | 'all';

type Props = {
  cluster: ClusterSummary | null;
  onClose: () => void;
};

export function ClusterAlertBoard({ cluster, onClose }: Props) {
  const [filter, setFilter] = useState<Filter>('all');

  useEffect(() => {
    if (!cluster) return;
    setFilter('all');
  }, [cluster?.name]);

  useEffect(() => {
    if (!cluster) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [cluster, onClose]);

  const alerts = useMemo(() => (cluster ? filterAlerts(cluster.alerts, filter) : []), [cluster, filter]);

  if (!cluster) return null;

  const { counts } = cluster;
  const total = counts.critical + counts.warning + counts.info;

  return (
    <div
      className="board-overlay"
      role="dialog"
      aria-modal="true"
      aria-label={`Алерты ${cluster.name}`}
      onClick={onClose}
    >
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
        </div>
        {alerts.length === 0 ? (
          <p style={{ color: 'var(--muted)', fontSize: 18 }}>Нет алертов в этой группе</p>
        ) : (
          <div className="alert-grid">
            {alerts.map((a) => (
              <article key={a.fingerprint} className={`alert-card sev-${a.severity}`}>
                <div className="meta">
                  <span className="sev">
                    <span className="dot" aria-hidden />
                    <span className="sev-label">{severityLabel(a.severity)}</span>
                  </span>
                  <span className="age mono">горит {firingDuration(a.startsAt)}</span>
                </div>
                <h3 className="title">{a.alertname}</h3>
                {a.summary ? <p className="summary">{a.summary}</p> : null}
                {a.description ? <p className="desc">{a.description}</p> : null}
                <div className="chips">
                  {alertChips(a).map((c) => (
                    <span key={`${c.key}-${c.value}`}>
                      {c.key}={c.value}
                    </span>
                  ))}
                </div>
                <div className="links">
                  {a.runbookUrl ? (
                    <a href={a.runbookUrl} target="_blank" rel="noreferrer">
                      Runbook
                    </a>
                  ) : null}
                  {a.generatorUrl ? (
                    <a href={a.generatorUrl} target="_blank" rel="noreferrer">
                      Источник
                    </a>
                  ) : null}
                </div>
              </article>
            ))}
          </div>
        )}
      </div>
    </div>
  );
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