import { snapshotTotals } from '../alerts';
import { foxQuote } from '../moodHysteresis';
import type { Mood, Snapshot } from '../types';

const assets: Record<Mood, string> = {
  calm: '/assets/fox-calm.jpg',
  worried: '/assets/fox-worried.jpg',
  panic: '/assets/fox-panic.jpg',
};

export function FoxBackdrop({ mood }: { mood: Mood }) {
  return (
    <div className="fox-backdrop" aria-hidden>
      {(Object.keys(assets) as Mood[]).map((m) => (
        <img key={m} className={m === mood ? 'active' : ''} src={assets[m]} alt="" />
      ))}
    </div>
  );
}

export function FoxStatus({ mood, snapshot }: { mood: Mood; snapshot: Snapshot | null }) {
  const totals = snapshotTotals(snapshot);
  const clusterN = snapshot?.clusters.length ?? 0;

  return (
    <div className="fox-status">
      <h2>{foxQuote(mood)}</h2>
      <p className="totals">
        {clusterN > 0 ? (
          <>
            Горит <b>{totals.clustersFiring}</b> из <b>{clusterN}</b> кластеров
          </>
        ) : (
          'Ждём данные с кластеров…'
        )}
      </p>
      <div className="stat-row">
        <span className="crit">crit {totals.critical}</span>
        <span className="warn">warn {totals.warning}</span>
        <span className="info">info {totals.info}</span>
      </div>
    </div>
  );
}