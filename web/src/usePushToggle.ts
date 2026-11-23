import { useCallback, useEffect, useState } from 'react';
import { urlBase64ToUint8Array, type PushToggleState } from './pushDevice';

export function usePushToggle() {
  const [state, setState] = useState<PushToggleState>('loading');

  useEffect(() => {
    let cancel = false;
    void (async () => {
      if (!('serviceWorker' in navigator) || !('PushManager' in window) || !('Notification' in window)) {
        if (!cancel) setState('unsupported');
        return;
      }
      try {
        const reg = await navigator.serviceWorker.ready;
        if (cancel) return;
        if (Notification.permission === 'denied') {
          setState('denied');
          return;
        }
        const existing = await reg.pushManager.getSubscription();
        if (cancel) return;
        if (existing) {
          setState('on');
          await putSubscription(existing);
          return;
        }
        setState('off');
      } catch {
        if (!cancel) setState('unsupported');
      }
    })();
    return () => {
      cancel = true;
    };
  }, []);

  const toggle = useCallback(async () => {
    if (state === 'denied' || state === 'unsupported' || state === 'loading') return;
    const reg = await navigator.serviceWorker.ready;
    if (state === 'on') {
      const existing = await reg.pushManager.getSubscription();
      if (existing) {
        const endpoint = existing.endpoint;
        await existing.unsubscribe();
        await deleteSubscription(endpoint);
      }
      setState('off');
      return;
    }
    const perm = await Notification.requestPermission();
    if (perm === 'denied') {
      setState('denied');
      return;
    }
    if (perm !== 'granted') return;
    const res = await fetch('/api/push/vapid', { credentials: 'same-origin' });
    if (!res.ok) return;
    const body = (await res.json()) as { publicKey?: string };
    if (!body.publicKey) return;
    const sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(body.publicKey),
    });
    await putSubscription(sub);
    setState('on');
  }, [state]);

  return { state, toggle };
}

async function putSubscription(sub: PushSubscription) {
  await fetch('/api/push/subscription', {
    method: 'PUT',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(sub),
  });
}

async function deleteSubscription(endpoint: string) {
  await fetch('/api/push/subscription', {
    method: 'DELETE',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ endpoint }),
  });
}
