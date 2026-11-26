export function amoVersion(runNumber: string, runAttempt: string): string;

export function setAddonVersion(manifestPath: string, runNumber: string, runAttempt: string): string;

export function packAddon(options: {
  sourceDir: string;
  keyPath: string;
  outDir: string;
}): Promise<{ crxPath: string; xpiPath: string }>;

export function prepareFirefoxSource(options: {
  sourceDir: string;
  outDir: string;
  runNumber: string;
  runAttempt: string;
}): Promise<void>;
