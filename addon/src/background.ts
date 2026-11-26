import { publishBadge } from './badgeAction';
import { mergeView } from './mergeView';
import { wallUrl } from './origin';
import { fetchSnapshot, interpretPoll, type FetchOutcome } from './poll';
import { loadSettings, loadSnapshots, loadViews, noticeUrl, rememberNotice, saveSnapshots, saveViews } from './storage';
import type { InstallView } from './types';

const POLL_ALARM = 'poll';
const POLL_PERIOD_MINUTES = 20 / 60;

function ensureAlarm(): void {
  void chrome.alarms.create(POLL_ALARM, { periodInMinutes: POLL_PERIOD_MINUTES });
}

let polling = false;
let queued = false;

function isFetchOutcome(value: unknown): value is FetchOutcome {
  if (!value || typeof value !== 'object') return false;
  const record = value as { error?: unknown; status?: unknown };
  return record.error === true || typeof record.status === 'number';
}

async function ensureOffscreen(): Promise<boolean> {
  if (!chrome.offscreen?.createDocument) return false;
  if (await chrome.offscreen.hasDocument()) return true;
  await chrome.offscreen.createDocument({
    url: 'offscreen.html',
    reasons: [chrome.offscreen.Reason.DOM_PARSER],
    justification: 'Read the alert wall from an extension page so the toolbar badge stays current',
  });
  return true;
}

async function pollOrigin(baseUrl: string): Promise<FetchOutcome> {
  try {
    if (await ensureOffscreen()) {
      for (let attempt = 0; attempt < 5; attempt++) {
        try {
          const outcome = await chrome.runtime.sendMessage({ type: 'poll-snapshot', baseUrl });
          if (isFetchOutcome(outcome)) return outcome;
        } catch {
          /* page is still starting */
        }
        await new Promise((resolve) => setTimeout(resolve, 50));
      }
    }
  } catch {
    /* offscreen is unavailable; the service worker fetches directly */
  }
  return fetchSnapshot(baseUrl);
}

async function pollOnce(): Promise<void> {
  const settings = await loadSettings();
  const remembered = await loadSnapshots();
  const stored = await loadViews();
  const views: Record<string, InstallView> = {};
  const known = new Set(settings.installations.map((install) => install.id));
  const results = new Map<string, ReturnType<typeof interpretPoll>>();

  for (const install of settings.installations) {
    if (!install.enabled) continue;
    const outcome = await pollOrigin(install.baseUrl);
    const result = interpretPoll(install, remembered[install.id] ?? null, outcome);
    results.set(install.id, result);
    if (result.remembered) remembered[install.id] = result.remembered;
    if (result.status !== 'ok') continue;
    for (const notice of result.notices) {
      await rememberNotice(notice.tag, wallUrl(install.baseUrl));
      await chrome.notifications.create(notice.tag, {
        type: 'basic',
        title: notice.title,
        message: notice.body,
        iconUrl: chrome.runtime.getURL('icons/icon-192.png'),
      });
    }
  }

  const latest = await loadViews();
  for (const [id, result] of results) {
    views[id] = mergeView(result, stored[id], latest[id]);
  }

  for (const id of Object.keys(remembered)) {
    if (!known.has(id)) delete remembered[id];
  }
  await saveSnapshots(remembered);
  await saveViews(views);
  await publishBadge(settings, views);
}

async function pollAll(): Promise<void> {
  if (polling) {
    queued = true;
    return;
  }
  polling = true;
  try {
    do {
      queued = false;
      await pollOnce();
    } while (queued);
  } finally {
    polling = false;
  }
}

chrome.runtime.onInstalled.addListener(() => {
  ensureAlarm();
  void pollAll();
});

chrome.runtime.onStartup.addListener(() => {
  ensureAlarm();
  void pollAll();
});

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === POLL_ALARM) void pollAll();
});

chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'sync' && changes.settings) void pollAll();
});

chrome.notifications.onClicked.addListener((id) => {
  void (async () => {
    const url = await noticeUrl(id);
    if (url) await chrome.tabs.create({ url });
    await chrome.notifications.clear(id);
  })();
});
