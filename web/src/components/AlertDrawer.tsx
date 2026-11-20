import type { ClusterSummary } from '../types';

type Props = {
  cluster: ClusterSummary | null;
  onClose: () => void;
};

export function AlertDrawer({ cluster, onClose }: Props) {
  if (!cluster) return null;
  return (
    <div className="drawer" role="dialog" onClick={onClose} onKeyDown={(e) => e.key === 'Escape' && onClose()}>
      <div className="card drawer-panel" onClick={(e) => e.stopPropagation()} role="document">
        <h2>{cluster.name}</h2>
        {cluster.alerts.length === 0 ? (
          <p style={{ color: 'var(--muted)' }}>Нет активных алертов</p>
        ) : (
          cluster.alerts.map((a) => (
            <div key={a.fingerprint} className="alert-row">
              <div className="sev">{a.severity}</div>
              <b>{a.alertname}</b>
              {a.namespace ? <span className="mono">{a.namespace}</span> : null}
              {a.summary ? <p>{a.summary}</p> : null}
              {a.runbookUrl ? (
                <a href={a.runbookUrl} target="_blank" rel="noreferrer">
                  Runbook
                </a>
              ) : null}
            </div>
          ))
        )}
        <button type="button" onClick={onClose} style={{ marginTop: 24 }}>
          Закрыть
        </button>
      </div>
    </div>
  );
}