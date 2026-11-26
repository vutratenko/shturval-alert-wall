import { diffSnapshots, noticesFor, type Notice } from './diff';
import { parseSnapshot } from './snapshot';
import type { Installation, Snapshot } from './types';

export type FetchOutcome =
  | { error: true }
  | { status: number; body: unknown; redirected?: boolean };

export type PollResult = {
  status: 'ok' | 'needs-login' | 'offline';
  snapshot: Snapshot | null;
  notices: Notice[];
  remembered: Snapshot | null;
};

export function interpretPoll(install: Installation, prev: Snapshot | null, response: FetchOutcome): PollResult {
  if ('error' in response) {
    return { status: 'offline', snapshot: prev, notices: [], remembered: prev };
  }
  if (response.redirected || response.status === 401 || response.status === 403 || (response.status >= 300 && response.status < 400)) {
    return { status: 'needs-login', snapshot: null, notices: [], remembered: prev };
  }
  if (response.status !== 200) {
    return { status: 'offline', snapshot: prev, notices: [], remembered: prev };
  }
  const snapshot = parseSnapshot(response.body);
  if (!snapshot) return { status: 'offline', snapshot: prev, notices: [], remembered: prev };
  return {
    status: 'ok',
    snapshot,
    notices: noticesFor(install, diffSnapshots(prev, snapshot)),
    remembered: snapshot,
  };
}

export async function fetchSnapshot(baseUrl: string, fetchImpl: typeof fetch = fetch): Promise<FetchOutcome> {
  try {
    // redirect:manual makes a service worker reject the request, so the badge stays "?"
    // while the popup, an extension page, reads the same wall just fine.
    const response = await fetchImpl(`${baseUrl}/api/snapshot`, {
      credentials: 'include',
    });
    if (response.type === 'opaqueredirect' || response.status === 0) {
      return { status: 401, body: null, redirected: true };
    }
    const text = await response.text();
    let body: unknown = null;
    if (text) {
      try {
        body = JSON.parse(text);
      } catch {
        body = null;
      }
    }
    return { status: response.status, body, redirected: false };
  } catch {
    return { error: true };
  }
}
