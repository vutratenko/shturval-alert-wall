import { badgeFor } from './badge';
import { originPattern, wallUrl } from './origin';
import { fetchSnapshot, interpretPoll } from './poll';
import { loadSettings, loadSnapshots, noticeUrl, rememberNotice, saveSnapshots, saveViews } from './storage';
import type { InstallView } from './types';

const POLL_ALARM = 'poll';
const POLL_PERIOD_MINUTES = 20 / 60;

function ensureAlarm(): void {
  void chrome.alarms.create(POLL_ALARM, { periodInMinutes: POLL_PERIOD_MINUTES });
}

let polling = false;
let queued = false;

async function pollOnce(): Promise<void> {
  const settings = await loadSettings();
  const remembered = await loadSnapshots();
  const views: Record<string, InstallView> = {};
  const known = new Set(settings.installations.map((install) => install.id));

  for (const install of settings.installations) {
    if (!install.enabled) continue;
    const allowed = await chrome.permissions.contains({ origins: [originPattern(install.baseUrl)] });
    const outcome = allowed ? await fetchSnapshot(install.baseUrl) : { error: true as const };
    const result = interpretPoll(install, remembered[install.id] ?? null, outcome);
    views[install.id] = {
      status: result.status,
      snapshot: result.status === 'ok' ? result.snapshot : null,
    };
    if (result.remembered) remembered[install.id] = result.remembered;
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

  for (const id of Object.keys(remembered)) {
    if (!known.has(id)) delete remembered[id];
  }
  await saveSnapshots(remembered);
  await saveViews(views);
  const badge = badgeFor(settings, views);
  await chrome.action.setBadgeText({ text: badge.text });
  await chrome.action.setBadgeBackgroundColor({ color: badge.color });
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
