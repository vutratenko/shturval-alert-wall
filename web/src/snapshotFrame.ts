import type { Snapshot, StreamEvent } from './types';

export function snapshotFromFrame(data: unknown): Snapshot | null {
  if (!data || typeof data !== 'object') return null;
  const frame = data as Snapshot & StreamEvent;
  if (Array.isArray(frame.clusters)) return frame;
  if (frame.snapshot && Array.isArray(frame.snapshot.clusters)) return frame.snapshot;
  return null;
}
