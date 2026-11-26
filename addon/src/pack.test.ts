import { generateKeyPairSync } from 'node:crypto';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';
import { afterEach, describe, expect, it } from 'vitest';
import { amoVersion, packAddon, prepareFirefoxSource, setAddonVersion } from '../pack.mjs';

const dirs: string[] = [];

afterEach(() => {
  for (const dir of dirs.splice(0)) rmSync(dir, { recursive: true, force: true });
});

function tempDir(): string {
  const dir = mkdtempSync(join(tmpdir(), 'alert-wall-pack-'));
  dirs.push(dir);
  return dir;
}

function fixture(dir: string): void {
  writeFileSync(
    join(dir, 'manifest.json'),
    JSON.stringify({
      manifest_version: 3,
      name: 'fixture',
      version: '0.1.0',
      browser_specific_settings: { gecko: { id: 'shturval-alert-wall@sion2k.ru' } },
    }),
  );
}

describe('packAddon', () => {
  it('writes a CRX3 and an XPI that names the Firefox add-on', async () => {
    const dir = tempDir();
    const source = join(dir, 'src');
    const out = join(dir, 'out');
    const keyPath = join(dir, 'key.pem');
    execFileSync('mkdir', ['-p', source]);
    fixture(source);
    const { privateKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
    writeFileSync(keyPath, privateKey.export({ type: 'pkcs8', format: 'pem' }));

    const packed = await packAddon({ sourceDir: source, keyPath, outDir: out });

    const crx = readFileSync(packed.crxPath);
    expect(crx.subarray(0, 4).toString('ascii')).toBe('Cr24');
    expect(crx.readUInt32LE(4)).toBe(3);
    const manifest = execFileSync('unzip', ['-p', packed.xpiPath, 'manifest.json'], { encoding: 'utf8' });
    expect(manifest).toContain('shturval-alert-wall@sion2k.ru');
    expect(readFileSync(keyPath, 'utf8')).toContain('PRIVATE KEY');
  });

  it('refuses to pack when the private key file is missing', async () => {
    const dir = tempDir();
    const source = join(dir, 'src');
    const keyPath = join(dir, 'missing.pem');
    execFileSync('mkdir', ['-p', source]);
    fixture(source);

    await expect(packAddon({ sourceDir: source, keyPath, outDir: join(dir, 'out') })).rejects.toThrow(
      /private key/,
    );
    expect(() => readFileSync(keyPath)).toThrow();
  });
});

describe('prepareFirefoxSource', () => {
  it('makes a classic background and a manifest Firefox can sign', async () => {
    const dir = tempDir();
    const source = join(dir, 'src');
    const chunks = join(source, 'chunks');
    execFileSync('mkdir', ['-p', chunks]);
    writeFileSync(join(chunks, 'mark.js'), 'export const mark = "wall";\n');
    writeFileSync(join(source, 'background.js'), 'import { mark } from "./chunks/mark.js";\nself.mark = mark;\n');
    writeFileSync(
      join(source, 'manifest.json'),
      JSON.stringify({
        manifest_version: 3,
        name: 'fixture',
        version: '0.1.0',
        optional_host_permissions: ['https://*/*', 'http://localhost/*', 'http://localhost:*/*'],
        background: { service_worker: 'background.js', type: 'module' },
        browser_specific_settings: { gecko: { id: 'shturval-alert-wall@sion2k.ru', strict_min_version: '128.0' } },
      }),
    );

    await prepareFirefoxSource({ sourceDir: source, outDir: join(dir, 'firefox'), runNumber: '7', runAttempt: '1' });

    const manifest = JSON.parse(readFileSync(join(dir, 'firefox', 'manifest.json'), 'utf8'));
    const background = readFileSync(join(dir, 'firefox', 'background.js'), 'utf8');
    expect(manifest.version).toBe('0.1.7.1');
    expect(manifest.optional_host_permissions).toEqual(['https://*/*', 'http://localhost/*']);
    expect(manifest.background.scripts).toEqual(['background.js']);
    expect(manifest.background.type).toBeUndefined();
    expect(manifest.browser_specific_settings.gecko.data_collection_permissions).toEqual({ required: ['none'] });
    expect(manifest.browser_specific_settings.gecko.strict_min_version).toBe('142.0');
    expect(background).not.toMatch(/\bimport\b/);
    expect(background).toContain('wall');
    expect(JSON.parse(readFileSync(join(source, 'manifest.json'), 'utf8')).version).toBe('0.1.0');
  });
});

describe('amoVersion', () => {
  it('uses the workflow run so AMO can sign each push', () => {
    expect(amoVersion('12', '1')).toBe('0.1.12.1');
    expect(amoVersion('12', '2')).toBe('0.1.12.2');
  });

  it('writes that version into a manifest copy', () => {
    const dir = tempDir();
    const manifestPath = join(dir, 'manifest.json');
    writeFileSync(manifestPath, JSON.stringify({ version: '0.1.0', name: 'fixture' }));

    setAddonVersion(manifestPath, '4', '1');

    expect(JSON.parse(readFileSync(manifestPath, 'utf8')).version).toBe('0.1.4.1');
  });
});
