import type { Mood } from '../types';
import { foxQuote } from '../moodHysteresis';

const assets: Record<Mood, string> = {
  calm: '/assets/fox-calm.jpg',
  worried: '/assets/fox-worried.jpg',
  panic: '/assets/fox-panic.jpg',
};

type Props = { mood: Mood };

export function Fox({ mood }: Props) {
  return (
    <div className={`card fox-bar fox-${mood}`}>
      <div className="fox-stage">
        {mood === 'calm' ? <span className="steam" aria-hidden /> : null}
        <img className="fox-img" src={assets[mood]} alt="" />
      </div>
      <div>
        <p className="fox-quote">
          <strong>Штурвал-лис:</strong> {foxQuote(mood)}
        </p>
      </div>
    </div>
  );
}