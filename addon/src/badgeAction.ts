import { badgeFor } from './badge';
import type { InstallView, Settings } from './types';

export async function publishBadge(settings: Settings, views: Record<string, InstallView>): Promise<void> {
  const badge = badgeFor(settings, views);
  await chrome.action.setBadgeText({ text: badge.text });
  await chrome.action.setBadgeBackgroundColor({ color: badge.color });
}
