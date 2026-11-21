import { describe, expect, it } from 'vitest';
import { wallScaleForViewport } from './wallScale';

describe('wallScaleForViewport', () => {
  it.each([
    [320, 568],
    [1199, 900],
  ])('uses responsive layout at %d×%d', (width, height) => {
    expect(wallScaleForViewport(width, height)).toBeNull();
  });

  it('switches to the scaled wall at 1200px', () => {
    expect(wallScaleForViewport(1200, 800)).toBeCloseTo(0.625);
  });

  it('keeps the native scale on a 1920×1080 viewport', () => {
    expect(wallScaleForViewport(1920, 1080)).toBe(1);
  });
});
