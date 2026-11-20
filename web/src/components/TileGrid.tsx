import { sortClusters } from '../tiles';
import type { ClusterSummary } from '../types';
import { ClusterTile } from './ClusterTile';

type Props = {
  clusters: ClusterSummary[];
  onSelect: (c: ClusterSummary) => void;
};

export function TileGrid({ clusters, onSelect }: Props) {
  const sorted = sortClusters(clusters);
  return (
    <div className="tile-grid">
      {sorted.map((c) => (
        <ClusterTile key={c.name} cluster={c} onSelect={() => onSelect(c)} />
      ))}
    </div>
  );
}