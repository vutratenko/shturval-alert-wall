const WALL_WIDTH = 1920;
const WALL_HEIGHT = 1080;
const RESPONSIVE_MAX_WIDTH = 1199;

export function wallScaleForViewport(width: number, height: number): number | null {
  if (width <= RESPONSIVE_MAX_WIDTH) return null;
  return Math.min(width / WALL_WIDTH, height / WALL_HEIGHT);
}
