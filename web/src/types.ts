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

export type Matcher = {
  name: string;
  value: string;
  isRegex?: boolean;
  isEqual?: boolean;
};

export type SilenceRef = {
  id: string;
  endsAt: string;
  createdBy?: string;
  comment?: string;
  matchers?: Matcher[];
};

export type SilencedAlert = Alert & { silences: SilenceRef[] };

export type Group = {
  id: string;
  keyKind: string;
  keyValue: string;
  severity: Bucket;
  startsAt: string;
  root: Alert;
  members: Alert[];
};

export type Counts = { critical: number; warning: number; info: number };

export type ClusterSummary = {
  name: string;
  source?: string;
  reachable: boolean;
  counts: Counts;
  alerts: Alert[];
  groups?: Group[];
  silenced?: SilencedAlert[];
};

export type Snapshot = {
  generatedAt: string;
  mood: Mood;
  silence?: boolean;
  clusters: ClusterSummary[];
};

export type StreamEvent = {
  type: string;
  cluster?: string;
  alert?: Alert;
  group?: Group;
  at?: string;
  snapshot?: Snapshot;
};