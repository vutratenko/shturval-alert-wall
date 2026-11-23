/// <reference lib="webworker" />
import { cleanupOutdatedCaches, matchPrecache, precacheAndRoute } from 'workbox-precaching';
import { isShellPath, leaveToNetwork } from './swPaths';

declare const self: ServiceWorkerGlobalScope;

// directoryIndex would serve the precached shell for "/" and fight the network-first handler.
precacheAndRoute(self.__WB_MANIFEST, { directoryIndex: '', cleanURLs: false });
cleanupOutdatedCaches();

self.addEventListener('install', (event) => {
  event.waitUntil(self.skipWaiting());
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);
  if (url.origin !== self.location.origin) return;
  if (leaveToNetwork(url.pathname)) return;
  if (event.request.mode !== 'navigate' || !isShellPath(url.pathname)) return;
  event.respondWith(networkFirstShell(event.request));
});

async function networkFirstShell(request: Request): Promise<Response> {
  try {
    const fresh = await fetch(request);
    if (fresh.ok) return fresh;
  } catch {
    /* offline */
  }
  const cached = (await matchPrecache('/index.html')) ?? (await matchPrecache('index.html'));
  if (cached) return cached;
  return new Response('offline', { status: 503, headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
}

interface PushBody {
  title?: string;
  body?: string;
  url?: string;
}

self.addEventListener('push', (event) => {
  const note = readPush(event);
  event.waitUntil(
    self.registration.showNotification(note.title, {
      body: note.body,
      tag: 'shturval-alerts',
      data: { url: note.url },
      icon: '/icons/icon-192.png',
    }),
  );
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const data = event.notification.data as { url?: string } | undefined;
  const target = new URL(data?.url || '/wall', self.location.origin).href;
  event.waitUntil(focusOrOpen(target));
});

function readPush(event: PushEvent): { title: string; body: string; url: string } {
  const fallback = { title: 'Штурвал', body: 'Новый алерт', url: '/wall' };
  if (!event.data) return fallback;
  try {
    const parsed = event.data.json() as PushBody;
    return {
      title: parsed.title || fallback.title,
      body: parsed.body || fallback.body,
      url: parsed.url || fallback.url,
    };
  } catch {
    return fallback;
  }
}

async function focusOrOpen(href: string): Promise<void> {
  const windows = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
  for (const client of windows) {
    if (new URL(client.url).origin !== self.location.origin) continue;
    await client.focus();
    return;
  }
  await self.clients.openWindow(href);
}
