import { describe, expect, it } from 'vitest';
import { isShellPath, leaveToNetwork } from './swPaths';

describe('service worker routing', () => {
  it('lets the browser handle the live stream, login, and metrics', () => {
    expect(leaveToNetwork('/api/stream')).toBe(true);
    expect(leaveToNetwork('/api/push/subscription')).toBe(true);
    expect(leaveToNetwork('/oauth2/login')).toBe(true);
    expect(leaveToNetwork('/metrics')).toBe(true);
    expect(leaveToNetwork('/metrics/extra')).toBe(true);
    expect(leaveToNetwork('/wall')).toBe(false);
  });

  it('treats the wall entrypoints as the app shell', () => {
    expect(isShellPath('/')).toBe(true);
    expect(isShellPath('/wall')).toBe(true);
    expect(isShellPath('/wall/')).toBe(true);
    expect(isShellPath('/oauth2/callback')).toBe(false);
    expect(isShellPath('/assets/app.js')).toBe(false);
  });
});
