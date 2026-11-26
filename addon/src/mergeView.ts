import type { PollResult } from './poll';
import type { InstallView } from './types';

function liveView(view: InstallView | undefined): InstallView | undefined {
  if (view?.status === 'ok' && view.snapshot) return view;
  return undefined;
}

export function mergeView(
  result: Pick<PollResult, 'status' | 'snapshot'>,
  prior: InstallView | undefined,
  latest: InstallView | undefined,
): InstallView {
  if (result.status === 'ok' && result.snapshot) {
    return { status: 'ok', snapshot: result.snapshot };
  }
  if (result.status === 'needs-login') {
    return { status: 'needs-login', snapshot: null };
  }
  return liveView(latest) ?? liveView(prior) ?? { status: 'offline', snapshot: null };
}
