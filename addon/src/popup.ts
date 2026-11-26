import { pickInstallation } from './badge';
import { loginUrl, wallUrl } from './origin';
import { parseSnapshot } from './snapshot';
import { setActive } from './settings';
import { loadSettings, loadViews, saveSettings } from './storage';
import { consumeSSE } from './stream';
import type { ClusterSummary, Installation, InstallView, Settings, Snapshot } from './types';
import './ui.css';

const MOOD_LABEL = { calm: 'спокойно', worried: 'тревога', panic: 'паника' } as const;

const installSelect = document.querySelector<HTMLSelectElement>('#install')!;
const empty = document.querySelector<HTMLParagraphElement>('#empty')!;
const statusEl = document.querySelector<HTMLParagraphElement>('#status')!;
const moodEl = document.querySelector<HTMLParagraphElement>('#mood')!;
const clustersEl = document.querySelector<HTMLUListElement>('#clusters')!;
const loginButton = document.querySelector<HTMLButtonElement>('#login')!;
const openButton = document.querySelector<HTMLButtonElement>('#open')!;
const optionsButton = document.querySelector<HTMLButtonElement>('#options')!;

let settings: Settings = { installations: [], activeId: null };
let views: Record<string, InstallView> = {};
let current: Installation | null = null;
let stream: AbortController | null = null;

function setStatus(text: string, needsLogin = false): void {
  statusEl.textContent = text;
  loginButton.hidden = !needsLogin;
}

function renderSnapshot(snapshot: Snapshot | null): void {
  clustersEl.replaceChildren();
  if (!snapshot) {
    moodEl.textContent = '';
    return;
  }
  const counts = snapshot.clusters.reduce(
    (sum, cluster) => ({
      critical: sum.critical + cluster.counts.critical,
      warning: sum.warning + cluster.counts.warning,
      info: sum.info + cluster.counts.info,
    }),
    { critical: 0, warning: 0, info: 0 },
  );
  moodEl.textContent = `${MOOD_LABEL[snapshot.mood]} · ${counts.critical} critical · ${counts.warning} warning · ${counts.info} info`;
  for (const cluster of snapshot.clusters) {
    clustersEl.append(clusterButton(cluster));
  }
}

function clusterButton(cluster: ClusterSummary): HTMLLIElement {
  const item = document.createElement('li');
  const button = document.createElement('button');
  button.type = 'button';
  button.className = 'cluster';
  const name = document.createElement('strong');
  name.textContent = cluster.name;
  const state = document.createElement('small');
  state.textContent = cluster.reachable ? 'на связи' : 'нет связи';
  if (!cluster.reachable) state.className = 'down';
  const counts = document.createElement('small');
  const critical = document.createElement('span');
  critical.className = 'crit';
  critical.textContent = String(cluster.counts.critical);
  const warning = document.createElement('span');
  warning.className = 'warn';
  warning.textContent = String(cluster.counts.warning);
  counts.append(
    critical,
    document.createTextNode(' · '),
    warning,
    document.createTextNode(` · ${cluster.counts.info}`),
  );
  button.append(name, state, counts);
  button.addEventListener('click', () => {
    if (current) void chrome.tabs.create({ url: wallUrl(current.baseUrl) });
  });
  item.append(button);
  return item;
}

function renderSelect(): void {
  installSelect.replaceChildren();
  const auto = document.createElement('option');
  auto.value = '';
  auto.textContent = 'Худшая из включённых';
  installSelect.append(auto);
  for (const install of settings.installations) {
    const option = document.createElement('option');
    option.value = install.id;
    option.textContent = install.enabled ? install.label : `${install.label} (выкл)`;
    installSelect.append(option);
  }
  installSelect.value = settings.activeId ?? '';
  installSelect.hidden = settings.installations.length === 0;
  empty.hidden = settings.installations.length > 0;
  openButton.hidden = settings.installations.length === 0;
}

function chosenInstall(): Installation | null {
  if (installSelect.value) {
    return settings.installations.find((install) => install.id === installSelect.value) ?? null;
  }
  return pickInstallation({ ...settings, activeId: null }, views);
}

function stopStream(): void {
  stream?.abort();
  stream = null;
}

async function watch(install: Installation): Promise<void> {
  stopStream();
  current = install;
  const controller = new AbortController();
  stream = controller;
  const cached = views[install.id];
  if (cached?.status === 'needs-login') setStatus('нужен вход', true);
  else if (cached?.status === 'ok') setStatus('live');
  else setStatus('offline');
  renderSnapshot(cached?.snapshot ?? null);
  try {
    const response = await fetch(`${install.baseUrl}/api/stream`, {
      credentials: 'include',
      redirect: 'manual',
      signal: controller.signal,
    });
    if (controller.signal.aborted || stream !== controller) return;
    if (
      response.type === 'opaqueredirect' ||
      response.status === 0 ||
      response.status === 401 ||
      response.status === 403 ||
      (response.status >= 300 && response.status < 400)
    ) {
      setStatus('нужен вход', true);
      renderSnapshot(null);
      return;
    }
    if (!response.ok) {
      setStatus('offline');
      return;
    }
    setStatus('live');
    await consumeSSE(response, (event) => {
      if (event.event !== 'snapshot' || stream !== controller) return;
      try {
        const snapshot = parseSnapshot(JSON.parse(event.data));
        if (snapshot) renderSnapshot(snapshot);
      } catch {
        /* ignore a broken frame */
      }
    });
    if (stream === controller) setStatus('offline');
  } catch (error) {
    if ((error as Error).name === 'AbortError' || stream !== controller) return;
    setStatus('offline');
  }
}

async function show(): Promise<void> {
  settings = await loadSettings();
  views = await loadViews();
  renderSelect();
  const install = chosenInstall();
  if (!install) {
    current = null;
    stopStream();
    setStatus('');
    renderSnapshot(null);
    return;
  }
  void watch(install);
}

installSelect.addEventListener('change', () => {
  void (async () => {
    settings = setActive(settings, installSelect.value || null);
    await saveSettings(settings);
    await show();
  })();
});

loginButton.addEventListener('click', () => {
  if (current) void chrome.tabs.create({ url: loginUrl(current.baseUrl) });
});

openButton.addEventListener('click', () => {
  if (current) void chrome.tabs.create({ url: wallUrl(current.baseUrl) });
});

optionsButton.addEventListener('click', () => {
  void chrome.runtime.openOptionsPage();
});

void show();
