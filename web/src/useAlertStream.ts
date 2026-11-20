import { useEffect, useRef, useState } from 'react';
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
        const data = JSON.parse(ev.data) as Snapshot & StreamEvent;
        if (Array.isArray(data.clusters)) {
          setSnapshot(data as Snapshot);
        } else if (data.snap) {
          setSnapshot(data.snap);
        }
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
    es.addEventListener('alert.firing', onEvent as EventListener);
    es.addEventListener('alert.resolved', onEvent as EventListener);
    es.addEventListener('cluster.down', onEvent as EventListener);
    es.addEventListener('cluster.up', onEvent as EventListener);
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