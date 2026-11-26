import { useEffect, useRef, useState } from 'react';
import { ClusterAlertBoard } from './components/ClusterAlertBoard';
import { EventFeed } from './components/EventFeed';
import { FoxBackdrop, FoxStatus } from './components/Fox';
import { PushToggle } from './components/PushToggle';
import { TileGrid } from './components/TileGrid';
import { nextMood, type HysteresisState } from './moodHysteresis';
import type { Mood } from './types';
import { useAlertStream } from './useAlertStream';
import { wallScaleForViewport } from './wallScale';

function useWallScale(ref: React.RefObject<HTMLDivElement | null>) {
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const fit = () => {
      const scale = wallScaleForViewport(window.innerWidth, window.innerHeight);
      if (scale === null) {
        el.style.removeProperty('transform');
      } else {
        el.style.transform = `scale(${scale})`;
      }
    };
    fit();
    window.addEventListener('resize', fit);
    return () => window.removeEventListener('resize', fit);
  }, [ref]);
}

function useFoxMood(raw: Mood | undefined): Mood {
  const [state, setState] = useState<HysteresisState>({ displayed: 'calm' });
  useEffect(() => {
    if (!raw) return;
    const id = window.setInterval(() => {
      setState((s) => nextMood(s, raw, Date.now()));
    }, 500);
    return () => window.clearInterval(id);
  }, [raw]);
  return state.displayed;
}

export function App() {
  const stageRef = useRef<HTMLDivElement>(null);
  useWallScale(stageRef);
  const { snapshot, events, live } = useAlertStream();
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const foxMood = useFoxMood(snapshot?.mood);
  const selected = snapshot?.clusters.find((cluster) => cluster.name === selectedName) ?? null;

  useEffect(() => {
    if (!selectedName || !snapshot) return;
    if (!snapshot.clusters.some((cluster) => cluster.name === selectedName)) setSelectedName(null);
  }, [snapshot, selectedName]);

  const timeLabel = snapshot?.generatedAt
    ? new Date(snapshot.generatedAt).toLocaleTimeString('ru-RU')
    : '—';

  return (
    <div className="wall-fit">
      <FoxBackdrop mood={foxMood} />
      <div ref={stageRef} className="wall-stage">
        <header className="wall-header">
          <div className="wall-brand">
            <img src="/assets/shturval-horizontal-on-dark.svg" alt="Штурвал" className="logo-wide" />
            <span className="eyebrow">Алерты</span>
          </div>
          <div className={`wall-live${live ? '' : ' offline'}`}>
            <PushToggle />
            <span className="dot" />
            <span className="mono">{live ? 'live' : 'offline'}</span>
            <span className="mono">{timeLabel}</span>
          </div>
        </header>
        <FoxStatus mood={foxMood} snapshot={snapshot} />
        <main className="wall-main">
          <TileGrid clusters={snapshot?.clusters ?? []} onSelect={(cluster) => setSelectedName(cluster.name)} />
          <EventFeed events={events} />
        </main>
        <ClusterAlertBoard
          cluster={selected}
          silenceEnabled={snapshot?.silence === true}
          onClose={() => setSelectedName(null)}
        />
      </div>
    </div>
  );
}
