import { createRequire } from 'node:module';
import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const require = createRequire(import.meta.url);
const crx3 = require('crx3');
const esbuild = require('esbuild');

const here = dirname(fileURLToPath(import.meta.url));

export function amoVersion(runNumber, runAttempt) {
  if (!/^[1-9][0-9]*$/.test(runNumber) || !/^[1-9][0-9]*$/.test(runAttempt)) {
    throw new Error('AMO version needs a positive run number and attempt');
  }
  return `0.1.${runNumber}.${runAttempt}`;
}

export function setAddonVersion(manifestPath, runNumber, runAttempt) {
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  manifest.version = amoVersion(runNumber, runAttempt);
  writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
  return manifest.version;
}

export async function prepareFirefoxSource({ sourceDir, outDir, runNumber, runAttempt }) {
  rmSync(outDir, { recursive: true, force: true });
  cpSync(sourceDir, outDir, { recursive: true });
  const manifestPath = join(outDir, 'manifest.json');
  const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
  manifest.version = amoVersion(runNumber, runAttempt);
  manifest.optional_host_permissions = (manifest.optional_host_permissions ?? []).filter(
    (pattern) => !pattern.includes(':*'),
  );
  const worker = manifest.background?.service_worker ?? 'background.js';
  manifest.background = { scripts: [worker], service_worker: worker };
  manifest.browser_specific_settings.gecko.strict_min_version = '142.0';
  manifest.browser_specific_settings.gecko.data_collection_permissions = { required: ['none'] };
  writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
  const bundled = join(outDir, `${worker}.bundled.js`);
  await esbuild.build({
    entryPoints: [join(outDir, worker)],
    bundle: true,
    format: 'iife',
    outfile: bundled,
    platform: 'browser',
  });
  rmSync(join(outDir, worker));
  cpSync(bundled, join(outDir, worker));
  rmSync(bundled);
}

export async function packAddon({ sourceDir, keyPath, outDir }) {
  if (!keyPath || !existsSync(keyPath)) {
    throw new Error(`CRX private key not found: ${keyPath || '(empty path)'}`);
  }
  mkdirSync(outDir, { recursive: true });
  const crxPath = join(outDir, 'shturval-alert-wall.crx');
  const xpiPath = join(outDir, 'shturval-alert-wall.xpi');
  await crx3([sourceDir], { keyPath, crxPath, zipPath: xpiPath });
  return { crxPath, xpiPath };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const keyPath = process.env.ADDON_CRX_KEY ?? '';
  if (!keyPath) {
    console.error('ADDON_CRX_KEY is required and must point at the existing CRX private key');
    process.exit(1);
  }
  const sourceDir = process.argv[2] ?? join(here, 'dist');
  const outDir = process.argv[3] ?? join(here, 'pack');
  packAddon({ sourceDir, keyPath, outDir }).catch((err) => {
    console.error(err instanceof Error ? err.message : err);
    process.exit(1);
  });
}
