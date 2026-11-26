import { normalizeOrigin, originPattern } from './origin';
import {
  addInstallation,
  removeInstallation,
  updateInstallation,
  type AddFailure,
} from './settings';
import { loadSettings, loadSnapshots, saveSettings, saveSnapshots } from './storage';
import type { Installation, NotifyLevel, Settings } from './types';
import './ui.css';

const REASONS: Record<AddFailure, string> = {
  'empty-label': 'Укажите название',
  'invalid-origin': 'Нужен адрес https://… или http://localhost',
  duplicate: 'Эта стена уже добавлена',
  'not-alert-wall': 'По этому адресу нет стены алертов Штурвала',
};

const NOTIFY: { value: NotifyLevel; label: string }[] = [
  { value: 'off', label: 'без алертов' },
  { value: 'critical', label: 'только critical' },
  { value: 'warning', label: 'critical и warning' },
  { value: 'all', label: 'все' },
];

const form = document.querySelector<HTMLFormElement>('#add')!;
const errorEl = document.querySelector<HTMLParagraphElement>('#error')!;
const list = document.querySelector<HTMLUListElement>('#list')!;

function showError(text: string): void {
  errorEl.textContent = text;
}

function render(settings: Settings): void {
  list.replaceChildren();
  for (const install of settings.installations) list.append(card(install));
}

function card(install: Installation): HTMLLIElement {
  const item = document.createElement('li');
  item.className = 'card';

  const header = document.createElement('header');
  const label = document.createElement('input');
  label.value = install.label;
  label.setAttribute('aria-label', 'Название');
  const remove = document.createElement('button');
  remove.type = 'button';
  remove.textContent = 'Удалить';
  header.append(label, remove);

  const url = document.createElement('code');
  url.textContent = install.baseUrl;

  const checks = document.createElement('div');
  checks.className = 'checks';
  const enabled = checkbox('Следить', install.enabled);
  const down = checkbox('Кластер недоступен', install.notifyClusterDown);
  const notify = document.createElement('select');
  notify.setAttribute('aria-label', 'Какие алерты присылать');
  for (const option of NOTIFY) {
    const itemOption = document.createElement('option');
    itemOption.value = option.value;
    itemOption.textContent = option.label;
    notify.append(itemOption);
  }
  notify.value = install.notify;
  checks.append(enabled.label, notify, down.label);

  label.addEventListener('change', () => {
    void patch(install.id, { label: label.value });
  });
  enabled.input.addEventListener('change', () => {
    void patch(install.id, { enabled: enabled.input.checked });
  });
  down.input.addEventListener('change', () => {
    void patch(install.id, { notifyClusterDown: down.input.checked });
  });
  notify.addEventListener('change', () => {
    void patch(install.id, { notify: notify.value as NotifyLevel });
  });
  remove.addEventListener('click', () => {
    void forget(install);
  });

  item.append(header, url, checks);
  return item;
}

function checkbox(text: string, checked: boolean): { label: HTMLLabelElement; input: HTMLInputElement } {
  const label = document.createElement('label');
  const input = document.createElement('input');
  input.type = 'checkbox';
  input.checked = checked;
  label.append(input, document.createTextNode(text));
  return { label, input };
}

async function patch(
  id: string,
  change: Partial<Pick<Installation, 'label' | 'enabled' | 'notify' | 'notifyClusterDown'>>,
): Promise<void> {
  const current = await loadSettings();
  await persist(updateInstallation(current, id, change));
}

async function persist(next: Settings): Promise<void> {
  await saveSettings(next);
  render(next);
}

async function forget(install: Installation): Promise<void> {
  const settings = await loadSettings();
  await saveSettings(removeInstallation(settings, install.id));
  await chrome.permissions.remove({ origins: [originPattern(install.baseUrl)] });
  const snapshots = await loadSnapshots();
  delete snapshots[install.id];
  await saveSnapshots(snapshots);
  render(await loadSettings());
}

form.addEventListener('submit', (event) => {
  event.preventDefault();
  showError('');
  const data = new FormData(form);
  const label = String(data.get('label') ?? '');
  const rawUrl = String(data.get('url') ?? '');
  const origin = normalizeOrigin(rawUrl);
  if (!label.trim()) {
    showError(REASONS['empty-label']);
    return;
  }
  if (!origin) {
    showError(REASONS['invalid-origin']);
    return;
  }
  const pattern = originPattern(origin);
  void chrome.permissions.request({ origins: [pattern] }).then(async (granted) => {
    if (!granted) {
      showError('Доступ к сайту не выдан');
      return;
    }
    let health: unknown = null;
    try {
      const response = await fetch(`${origin}/api/healthz`, { credentials: 'include', redirect: 'manual' });
      health = await response.json();
    } catch {
      health = null;
    }
    const settings = await loadSettings();
    const added = addInstallation(settings, { id: crypto.randomUUID(), label, baseUrl: origin }, health);
    if (!added.ok) {
      if (added.reason !== 'duplicate') await chrome.permissions.remove({ origins: [pattern] });
      showError(REASONS[added.reason]);
      return;
    }
    await saveSettings(added.settings);
    form.reset();
    render(added.settings);
  });
});

void loadSettings().then(render);
