import { useEffect, useRef, useState } from 'react';
import { snapshotFromFrame } from './snapshotFrame';
import type { Snapshot, StreamEvent } from './types';

export function useAlertStream() {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [events, setEvents] = useState<StreamEvent[]>([]);
  const [live, setLive] = useState(false);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const es = new EventSource('/api/stream');
    esRef.current = es;

    const onSnapshot = (ev: MessageEvent) => {
      setLive(true);
      try {
        const next = snapshotFromFrame(JSON.parse(ev.data));
        if (next) setSnapshot(next);
      } catch {
        /* ignore */
      }
    };
    const onEvent = (ev: MessageEvent) => {
      setLive(true);
      try {
        const parsed = JSON.parse(ev.data) as StreamEvent;
        setEvents((prev) => [parsed, ...prev].slice(0, 80));
      } catch {
        /* ignore */
      }
    };

    es.addEventListener('snapshot', onSnapshot as EventListener);
    for (const name of ['alert.firing', 'alert.resolved', 'alert.silenced', 'alert.unsilenced', 'cluster.down', 'cluster.up']) {
      es.addEventListener(name, onEvent as EventListener);
    }
    let loginStarted = false;
    es.onerror = () => {
      setLive(false);
      if (loginStarted) return;
      void (async () => {
        try {
          const res = await fetch('/api/snapshot', { credentials: 'same-origin' });
          if (res.status === 401 && !loginStarted) {
            loginStarted = true;
            es.close();
            const rd = window.location.pathname + window.location.search;
            window.location.assign(`/oauth2/login?rd=${encodeURIComponent(rd)}`);
          }
        } catch {
          /* ignore */
        }
      })();
    };

    return () => {
      es.close();
      esRef.current = null;
    };
  }, []);

  return { snapshot, events, live };
}