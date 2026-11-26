import { parseSettings } from './settings';
import type { InstallView, Settings, Snapshot } from './types';

const SETTINGS = 'settings';
const SNAPSHOTS = 'snapshots';
const VIEWS = 'views';
const NOTICE_URLS = 'noticeUrls';

export async function loadSettings(): Promise<Settings> {
  const bag = await chrome.storage.sync.get(SETTINGS);
  return parseSettings(bag[SETTINGS]);
}

export async function saveSettings(settings: Settings): Promise<void> {
  await chrome.storage.sync.set({ [SETTINGS]: settings });
}

export async function loadSnapshots(): Promise<Record<string, Snapshot>> {
  const bag = await chrome.storage.session.get(SNAPSHOTS);
  const raw = bag[SNAPSHOTS];
  if (!raw || typeof raw !== 'object') return {};
  const snapshots: Record<string, Snapshot> = {};
  for (const [id, value] of Object.entries(raw as Record<string, unknown>)) {
    if (value && typeof value === 'object' && Array.isArray((value as Snapshot).clusters)) {
      snapshots[id] = value as Snapshot;
    }
  }
  return snapshots;
}

export async function saveSnapshots(snapshots: Record<string, Snapshot>): Promise<void> {
  await chrome.storage.session.set({ [SNAPSHOTS]: snapshots });
}

export async function loadViews(): Promise<Record<string, InstallView>> {
  const bag = await chrome.storage.session.get(VIEWS);
  const raw = bag[VIEWS];
  if (!raw || typeof raw !== 'object') return {};
  return raw as Record<string, InstallView>;
}

export async function saveViews(views: Record<string, InstallView>): Promise<void> {
  await chrome.storage.session.set({ [VIEWS]: views });
}

export async function rememberNotice(tag: string, url: string): Promise<void> {
  const bag = await chrome.storage.session.get(NOTICE_URLS);
  const current = bag[NOTICE_URLS];
  const urls: Record<string, string> = current && typeof current === 'object' ? { ...(current as Record<string, string>) } : {};
  urls[tag] = url;
  await chrome.storage.session.set({ [NOTICE_URLS]: urls });
}

export async function noticeUrl(tag: string): Promise<string | null> {
  const bag = await chrome.storage.session.get(NOTICE_URLS);
  const urls = bag[NOTICE_URLS];
  if (!urls || typeof urls !== 'object') return null;
  const url = (urls as Record<string, string>)[tag];
  return typeof url === 'string' ? url : null;
}
