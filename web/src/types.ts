export type Mood = 'calm' | 'worried' | 'panic';

export type Bucket = 'critical' | 'warning' | 'info';

export type Alert = {
  fingerprint: string;
  alertname: string;
  severity: Bucket;
  namespace?: string;
  cluster?: string;
  summary?: string;
  description?: string;
  runbookUrl?: string;
  generatorUrl?: string;
  labels?: Record<string, string>;
  startsAt: string;
};

export type Counts = { critical: number; warning: number; info: number };

export type ClusterSummary = {
  name: string;
  reachable: boolean;
  counts: Counts;
  alerts: Alert[];
};

export type Snapshot = {
  generatedAt: string;
  mood: Mood;
  clusters: ClusterSummary[];
};

export type StreamEvent = {
  type: string;
  cluster?: string;
  alert?: Alert;
  at?: string;
  snapshot?: Snapshot;
};