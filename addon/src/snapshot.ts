import type { Snapshot } from './types';

export function parseSnapshot(data: unknown): Snapshot | null {
  if (!data || typeof data !== 'object') return null;
  const frame = data as { clusters?: unknown; snapshot?: unknown };
  if (Array.isArray(frame.clusters)) return frame as Snapshot;
  if (frame.snapshot && typeof frame.snapshot === 'object' && Array.isArray((frame.snapshot as Snapshot).clusters)) {
    return frame.snapshot as Snapshot;
  }
  return null;
}

export type SSEEvent = { event: string; data: string };

export function takeSSEEvents(buffer: string): { events: SSEEvent[]; rest: string } {
  const parts = buffer.split('\n\n');
  const rest = parts.pop() ?? '';
  const events: SSEEvent[] = [];
  for (const part of parts) {
    let event = 'message';
    const dataLines: string[] = [];
    for (const line of part.split('\n')) {
      if (!line || line.startsWith(':')) continue;
      if (line.startsWith('event:')) event = line.slice(6).trim();
      else if (line.startsWith('data:')) dataLines.push(line.slice(5).replace(/^ /, ''));
    }
    if (dataLines.length) events.push({ event, data: dataLines.join('\n') });
  }
  return { events, rest };
}
