import { normalizeOrigin } from './origin';
import type { Installation, NotifyLevel, Settings } from './types';

export const ALERT_WALL_SERVICE = 'shturval-alert-wall';

const NOTIFY_LEVELS = new Set<NotifyLevel>(['off', 'critical', 'warning', 'all']);

export type AddFailure = 'empty-label' | 'invalid-origin' | 'duplicate' | 'not-alert-wall';

export function emptySettings(): Settings {
  return { installations: [], activeId: null };
}

export function isAlertWallHealth(body: unknown): boolean {
  if (!body || typeof body !== 'object') return false;
  const record = body as Record<string, unknown>;
  return record.status === 'ok' && record.service === ALERT_WALL_SERVICE;
}

function parseInstallation(item: unknown): Installation | null {
  if (!item || typeof item !== 'object') return null;
  const record = item as Record<string, unknown>;
  if (typeof record.id !== 'string' || !record.id) return null;
  if (typeof record.label !== 'string' || !record.label.trim()) return null;
  if (typeof record.baseUrl !== 'string') return null;
  const baseUrl = normalizeOrigin(record.baseUrl);
  if (!baseUrl) return null;
  const notify = typeof record.notify === 'string' && NOTIFY_LEVELS.has(record.notify as NotifyLevel)
    ? (record.notify as NotifyLevel)
    : 'critical';
  return {
    id: record.id,
    label: record.label.trim(),
    baseUrl,
    enabled: record.enabled !== false,
    notify,
    notifyClusterDown: record.notifyClusterDown !== false,
  };
}

export function parseSettings(raw: unknown): Settings {
  if (!raw || typeof raw !== 'object') return emptySettings();
  const record = raw as Record<string, unknown>;
  const installations: Installation[] = [];
  if (Array.isArray(record.installations)) {
    for (const item of record.installations) {
      const install = parseInstallation(item);
      if (!install) continue;
      if (installations.some((existing) => existing.id === install.id || existing.baseUrl === install.baseUrl)) {
        continue;
      }
      installations.push(install);
    }
  }
  const activeId = typeof record.activeId === 'string' && installations.some((install) => install.id === record.activeId)
    ? record.activeId
    : null;
  return { installations, activeId };
}

export function addInstallation(
  settings: Settings,
  draft: { id: string; label: string; baseUrl: string },
  health: unknown,
): { ok: true; settings: Settings } | { ok: false; reason: AddFailure } {
  const label = draft.label.trim();
  if (!label) return { ok: false, reason: 'empty-label' };
  const baseUrl = normalizeOrigin(draft.baseUrl);
  if (!baseUrl) return { ok: false, reason: 'invalid-origin' };
  if (!isAlertWallHealth(health)) return { ok: false, reason: 'not-alert-wall' };
  if (settings.installations.some((install) => install.baseUrl === baseUrl)) {
    return { ok: false, reason: 'duplicate' };
  }
  const install: Installation = {
    id: draft.id,
    label,
    baseUrl,
    enabled: true,
    notify: 'critical',
    notifyClusterDown: true,
  };
  return {
    ok: true,
    settings: {
      installations: [...settings.installations, install],
      activeId: settings.activeId ?? install.id,
    },
  };
}

export function updateInstallation(
  settings: Settings,
  id: string,
  patch: Partial<Pick<Installation, 'label' | 'enabled' | 'notify' | 'notifyClusterDown'>>,
): Settings {
  return {
    ...settings,
    installations: settings.installations.map((install) => {
      if (install.id !== id) return install;
      const label = patch.label !== undefined ? patch.label.trim() : install.label;
      return {
        ...install,
        label: label || install.label,
        enabled: patch.enabled ?? install.enabled,
        notify: patch.notify ?? install.notify,
        notifyClusterDown: patch.notifyClusterDown ?? install.notifyClusterDown,
      };
    }),
  };
}

export function removeInstallation(settings: Settings, id: string): Settings {
  return {
    installations: settings.installations.filter((install) => install.id !== id),
    activeId: settings.activeId === id ? null : settings.activeId,
  };
}

export function setActive(settings: Settings, id: string | null): Settings {
  if (id !== null && !settings.installations.some((install) => install.id === id)) return settings;
  return { ...settings, activeId: id };
}
