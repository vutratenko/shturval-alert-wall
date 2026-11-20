import { clusterTotal, tileTone } from '../tiles';
import type { ClusterSummary } from '../types';

type Props = {
  cluster: ClusterSummary;
  onSelect: () => void;
};

export function ClusterTile({ cluster, onSelect }: Props) {
  const tone = tileTone(cluster);
  const total = clusterTotal(cluster);
  return (
    <button type="button" className={`card cluster-tile ${tone}`} onClick={onSelect}>
      <div className="name">{cluster.reachable ? cluster.name : `${cluster.name} · нет связи`}</div>
      <div className="total">{cluster.reachable ? total : '—'}</div>
      <div className="counts">
        <span>crit {cluster.counts.critical}</span>
        <span>warn {cluster.counts.warning}</span>
        <span>info {cluster.counts.info}</span>
      </div>
    </button>
  );
}