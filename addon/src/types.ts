export type Mood = 'calm' | 'worried' | 'panic';

export type Bucket = 'critical' | 'warning' | 'info';

export type NotifyLevel = 'off' | 'critical' | 'warning' | 'all';

export type Alert = {
  fingerprint: string;
  alertname: string;
  severity: Bucket;
  cluster?: string;
};

export type Counts = { critical: number; warning: number; info: number };

export type ClusterSummary = {
  name: string;
  reachable: boolean;
  counts: Counts;
  alerts: Alert[];
  silenced?: Alert[];
};

export type Snapshot = {
  generatedAt: string;
  mood: Mood;
  clusters: ClusterSummary[];
};

export type Installation = {
  id: string;
  label: string;
  baseUrl: string;
  enabled: boolean;
  notify: NotifyLevel;
  notifyClusterDown: boolean;
};

export type Settings = {
  installations: Installation[];
  activeId: string | null;
};

export type InstallStatus = 'ok' | 'needs-login' | 'offline';

export type InstallView = {
  status: InstallStatus;
  snapshot: Snapshot | null;
};
