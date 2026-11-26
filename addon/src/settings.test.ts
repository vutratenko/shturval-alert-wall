import { describe, expect, it } from 'vitest';
import {
  addInstallation,
  emptySettings,
  isAlertWallHealth,
  parseSettings,
  removeInstallation,
  setActive,
  updateInstallation,
} from './settings';

const health = { status: 'ok', service: 'shturval-alert-wall' };

describe('isAlertWallHealth', () => {
  it('accepts only this service', () => {
    expect(isAlertWallHealth(health)).toBe(true);
    expect(isAlertWallHealth({ status: 'ok' })).toBe(false);
    expect(isAlertWallHealth({ status: 'ok', service: 'other' })).toBe(false);
    expect(isAlertWallHealth(null)).toBe(false);
  });
});

describe('installations', () => {
  it('adds a wall, rejects a bad one, and selects the first', () => {
    const added = addInstallation(emptySettings(), {
      id: '1',
      label: ' prod ',
      baseUrl: 'https://alerts.example.com/wall',
    }, health);
    expect(added).toMatchObject({
      ok: true,
      settings: {
        activeId: '1',
        installations: [{ id: '1', label: 'prod', baseUrl: 'https://alerts.example.com', notify: 'critical', notifyClusterDown: true, enabled: true }],
      },
    });
    if (!added.ok) return;
    expect(addInstallation(added.settings, { id: '2', label: 'lab', baseUrl: 'https://alerts.example.com' }, health)).toEqual({
      ok: false,
      reason: 'duplicate',
    });
    expect(addInstallation(emptySettings(), { id: '2', label: '  ', baseUrl: 'https://alerts.example.com' }, health).ok).toBe(false);
    expect(addInstallation(emptySettings(), { id: '2', label: 'lab', baseUrl: 'http://alerts.example.com' }, health)).toMatchObject({
      reason: 'invalid-origin',
    });
    expect(addInstallation(emptySettings(), { id: '2', label: 'lab', baseUrl: 'https://alerts.example.com' }, { status: 'ok' })).toMatchObject({
      reason: 'not-alert-wall',
    });
  });

  it('updates, removes, and keeps a second wall active', () => {
    const first = addInstallation(emptySettings(), { id: '1', label: 'prod', baseUrl: 'https://a.example' }, health);
    if (!first.ok) throw new Error('add');
    const second = addInstallation(first.settings, { id: '2', label: 'lab', baseUrl: 'https://b.example' }, health);
    if (!second.ok) throw new Error('add');
    expect(second.settings.activeId).toBe('1');
    const renamed = updateInstallation(second.settings, '2', { label: '  lab 2 ', enabled: false, notify: 'all', notifyClusterDown: false });
    expect(renamed.installations[1]).toMatchObject({ label: 'lab 2', enabled: false, notify: 'all', notifyClusterDown: false });
    expect(updateInstallation(renamed, '2', { label: '   ' }).installations[1]?.label).toBe('lab 2');
    expect(removeInstallation(renamed, '1').activeId).toBeNull();
    expect(setActive(second.settings, '2').activeId).toBe('2');
    expect(setActive(second.settings, 'missing').activeId).toBe('1');
  });

  it('drops garbage and duplicate origins when reading storage', () => {
    expect(parseSettings(null)).toEqual(emptySettings());
    const parsed = parseSettings({
      activeId: 'gone',
      installations: [
        { id: '1', label: ' prod ', baseUrl: 'https://a.example/wall', notify: 'nope' },
        { id: '1', label: 'dup', baseUrl: 'https://b.example' },
        { id: '2', label: 'same', baseUrl: 'https://a.example' },
        { label: 'missing id' },
        { id: '3', label: 'http', baseUrl: 'http://a.example' },
      ],
    });
    expect(parsed.installations).toEqual([
      {
        id: '1',
        label: 'prod',
        baseUrl: 'https://a.example',
        enabled: true,
        notify: 'critical',
        notifyClusterDown: true,
      },
    ]);
    expect(parsed.activeId).toBeNull();
  });
});
