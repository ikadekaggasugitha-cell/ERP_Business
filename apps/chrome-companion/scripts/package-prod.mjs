#!/usr/bin/env node
// Production packager & MV3 validator for ERP Enterprise Companion.
// Usage:
//   node scripts/package-prod.mjs --dry-run
//   node scripts/package-prod.mjs --origin https://erp.sinergi.example.com/* --out dist-prod

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const EXTENSION_ROOT = path.resolve(__dirname, '..');

const DEFAULT_ORIGIN = 'https://erp.sinergi.example.com/*';
const LOCALHOST_PATTERN = /^https?:\/\/(localhost|127\.0\.0\.1|\[::1\])(?::\d+)?(\/.*)?$/i;
const PLACEHOLDER_ERP_PATTERN = /^https:\/\/(?:\*\.|api\.)?erp\.example(?:\/.*)?$/i;

const RUNTIME_ENTRIES = [
  'service-worker.js',
  'sidepanel',
  'content',
  'sandbox',
  'icons',
];

function parseArgs(argv) {
  const args = {
    origin: DEFAULT_ORIGIN,
    dryRun: false,
    outDir: path.join(EXTENSION_ROOT, 'dist-prod'),
  };

  for (let i = 0; i < argv.length; i += 1) {
    const token = argv[i];
    if (token === '--dry-run') {
      args.dryRun = true;
    } else if (token === '--origin' && argv[i + 1]) {
      args.origin = argv[++i];
    } else if (token.startsWith('--origin=')) {
      args.origin = token.slice('--origin='.length);
    } else if (token === '--out' && argv[i + 1]) {
      args.outDir = path.resolve(argv[++i]);
    } else if (token.startsWith('--out=')) {
      args.outDir = path.resolve(token.slice('--out='.length));
    } else if (token === '--help' || token === '-h') {
      console.log(
        'Usage: node scripts/package-prod.mjs [--origin <https://.../*>] [--out <dir>] [--dry-run]'
      );
      process.exit(0);
    } else {
      throw new Error(`Unknown argument: ${token}`);
    }
  }

  return args;
}

function normalizeOriginPattern(rawOrigin) {
  const trimmed = String(rawOrigin || '').trim();
  if (!trimmed) {
    throw new Error('--origin must not be empty');
  }
  const withoutWildcard = trimmed.endsWith('/*') ? trimmed.slice(0, -2) : trimmed.replace(/\/+$/, '');
  let parsed;
  try {
    parsed = new URL(withoutWildcard);
  } catch {
    throw new Error(`Invalid --origin URL: ${rawOrigin}`);
  }
  if (parsed.protocol !== 'https:') {
    throw new Error(`Production --origin must use https:// (got ${parsed.protocol})`);
  }
  if (LOCALHOST_PATTERN.test(parsed.origin)) {
    throw new Error(`Production --origin must not target localhost (got ${rawOrigin})`);
  }
  return `${parsed.origin}/*`;
}

function transformManifest(rawManifest, originPattern) {
  const manifest = structuredClone(rawManifest);
  const strippedHostPermissions = [];
  const strippedContentMatches = [];

  const nextHostPerms = [];
  for (const perm of manifest.host_permissions || []) {
    if (LOCALHOST_PATTERN.test(perm)) {
      strippedHostPermissions.push(perm);
      continue;
    }
    if (PLACEHOLDER_ERP_PATTERN.test(perm)) {
      if (!nextHostPerms.includes(originPattern)) {
        nextHostPerms.push(originPattern);
      }
      continue;
    }
    if (!nextHostPerms.includes(perm)) {
      nextHostPerms.push(perm);
    }
  }
  if (!nextHostPerms.includes(originPattern)) {
    nextHostPerms.unshift(originPattern);
  }
  manifest.host_permissions = nextHostPerms;

  if (Array.isArray(manifest.content_scripts)) {
    manifest.content_scripts = manifest.content_scripts.map((cs) => {
      const nextMatches = [];
      for (const match of cs.matches || []) {
        if (LOCALHOST_PATTERN.test(match)) {
          strippedContentMatches.push(match);
          continue;
        }
        if (PLACEHOLDER_ERP_PATTERN.test(match)) {
          if (!nextMatches.includes(originPattern)) {
            nextMatches.push(originPattern);
          }
          continue;
        }
        if (!nextMatches.includes(match)) {
          nextMatches.push(match);
        }
      }
      return { ...cs, matches: nextMatches };
    });
  }

  return {
    manifest,
    strippedHostPermissions,
    strippedContentMatches,
  };
}

function validateManifest(manifest, rootDir) {
  const errors = [];

  if (manifest.manifest_version !== 3) {
    errors.push(`manifest_version must be 3 (got ${manifest.manifest_version})`);
  }
  for (const field of ['name', 'version', 'description']) {
    if (!manifest[field] || String(manifest[field]).trim() === '') {
      errors.push(`Required MV3 field missing or empty: ${field}`);
    }
  }

  const swPath = manifest.background?.service_worker;
  if (!swPath) {
    errors.push('background.service_worker is required');
  } else if (!fs.existsSync(path.join(rootDir, swPath))) {
    errors.push(`background.service_worker file not found: ${swPath}`);
  }

  const spPath = manifest.side_panel?.default_path;
  if (!spPath) {
    errors.push('side_panel.default_path is required');
  } else if (!fs.existsSync(path.join(rootDir, spPath))) {
    errors.push(`side_panel.default_path file not found: ${spPath}`);
  }

  for (const size of ['16', '48', '128']) {
    const iconRel = manifest.icons?.[size];
    if (!iconRel) {
      errors.push(`icons.${size} is required`);
    } else if (!fs.existsSync(path.join(rootDir, iconRel))) {
      errors.push(`Icon file not found: ${iconRel}`);
    }
  }

  for (const perm of manifest.host_permissions || []) {
    if (LOCALHOST_PATTERN.test(perm)) {
      errors.push(`Localhost host_permission leaked into production manifest: ${perm}`);
    }
  }

  if (!Array.isArray(manifest.content_scripts) || manifest.content_scripts.length === 0) {
    errors.push('content_scripts must be a non-empty array');
  } else {
    for (const cs of manifest.content_scripts) {
      if (cs.all_frames !== true) {
        errors.push(`content_script (${(cs.js || []).join(',')}) must set all_frames: true`);
      }
      if (!Array.isArray(cs.matches) || cs.matches.length === 0) {
        errors.push(`content_script (${(cs.js || []).join(',')}) has empty matches`);
      }
      for (const m of cs.matches || []) {
        if (LOCALHOST_PATTERN.test(m)) {
          errors.push(`Localhost content_script match leaked into production manifest: ${m}`);
        }
      }
      for (const jsFile of cs.js || []) {
        if (!fs.existsSync(path.join(rootDir, jsFile))) {
          errors.push(`content_script file not found: ${jsFile}`);
        }
      }
    }
  }

  for (const page of manifest.sandbox?.pages || []) {
    if (!fs.existsSync(path.join(rootDir, page))) {
      errors.push(`sandbox page not found: ${page}`);
    }
  }

  return errors;
}

function copyRuntimeFiles(srcRoot, outDir, prodManifest) {
  fs.rmSync(outDir, { recursive: true, force: true });
  fs.mkdirSync(outDir, { recursive: true });

  const copied = [];
  for (const entry of RUNTIME_ENTRIES) {
    const srcPath = path.join(srcRoot, entry);
    const dstPath = path.join(outDir, entry);
    if (!fs.existsSync(srcPath)) continue;
    fs.cpSync(srcPath, dstPath, { recursive: true });
    copied.push(entry);
  }

  fs.writeFileSync(
    path.join(outDir, 'manifest.json'),
    `${JSON.stringify(prodManifest, null, 2)}\n`,
    'utf8'
  );
  copied.push('manifest.json');
  return copied;
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  const originPattern = normalizeOriginPattern(args.origin);

  const manifestPath = path.join(EXTENSION_ROOT, 'manifest.json');
  const rawManifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  const { manifest, strippedHostPermissions, strippedContentMatches } = transformManifest(
    rawManifest,
    originPattern
  );

  const validationErrors = validateManifest(manifest, EXTENSION_ROOT);
  if (validationErrors.length > 0) {
    console.error('[package-prod] MV3 manifest validation FAILED:');
    for (const err of validationErrors) {
      console.error(`  - ${err}`);
    }
    process.exit(1);
  }

  const summary = {
    mode: args.dryRun ? 'dry-run' : 'build',
    origin: originPattern,
    outDir: args.dryRun ? null : args.outDir,
    manifestVersion: manifest.manifest_version,
    version: manifest.version,
    strippedHostPermissions,
    strippedContentMatches,
    hostPermissions: manifest.host_permissions,
    runtimeEntries: [...RUNTIME_ENTRIES, 'manifest.json'],
    valid: true,
  };

  if (!args.dryRun) {
    summary.copiedEntries = copyRuntimeFiles(EXTENSION_ROOT, args.outDir, manifest);
  }

  console.log(JSON.stringify(summary, null, 2));
}

main();
