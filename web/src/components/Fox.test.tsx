import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { Mood } from '../types';
import { FoxBackdrop } from './Fox';

const moods: Mood[] = ['calm', 'worried', 'panic'];

describe('FoxBackdrop', () => {
  for (const mood of moods) {
    it(`keeps the ${mood} character and glow in sync`, () => {
      const html = renderToStaticMarkup(<FoxBackdrop mood={mood} />);

      expect(html).toContain(`class="mood-glow ${mood} active"`);
      expect(html.match(/\bactive\b/g)).toHaveLength(2);
      expect(html).toContain(`src="/assets/fox-${mood}.jpg"`);
    });
  }
});
