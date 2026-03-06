#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

function walk(dir, exts, fileList = []) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const e of entries) {
    const res = path.resolve(dir, e.name);
    if (e.isDirectory()) walk(res, exts, fileList);
    else if (exts.includes(path.extname(e.name))) fileList.push(res);
  }
  return fileList;
}

function collectKeysFromCode() {
  const files = walk(path.resolve(process.cwd(), 'src'), ['.js', '.jsx', '.ts', '.tsx', '.md']);
  const keyRe = /\bt\(\s*['"`]([^'"`]+)['"`]\s*[,)\}]/g;
  const used = new Set();
  for (const f of files) {
    const text = fs.readFileSync(f, 'utf8');
    let m;
    while ((m = keyRe.exec(text))) used.add(m[1]);
  }
  return used;
}

function collectLocaleKeys() {
  const localesDir = path.resolve(process.cwd(), 'public', 'locales');
  const langs = fs.existsSync(localesDir) ? fs.readdirSync(localesDir) : [];
  const localeMap = {}; // lang -> Set(keys)
  for (const lang of langs) {
    const files = walk(path.join(localesDir, lang), ['.json']);
    const set = new Set();
    for (const f of files) {
      try {
        const json = JSON.parse(fs.readFileSync(f, 'utf8'));
        function flatten(obj, prefix = '') {
          for (const k of Object.keys(obj)) {
            const val = obj[k];
            const key = prefix ? `${prefix}.${k}` : k;
            if (typeof val === 'string') set.add(key);
            else if (typeof val === 'object' && val !== null) flatten(val, key);
          }
        }
        flatten(json);
      } catch (e) {
        console.error(`Failed to parse ${f}: ${e.message}`);
        process.exitCode = 2;
        return null;
      }
    }
    localeMap[lang] = set;
  }
  return localeMap;
}

function run() {
  console.log('Collecting keys from source...');
  const used = collectKeysFromCode();
  console.log(`Found ${used.size} translation keys in source.`);

  console.log('Collecting locale keys...');
  const localeMap = collectLocaleKeys();
  if (!localeMap) return;
  const langs = Object.keys(localeMap);
  if (langs.length === 0) {
    console.warn('No locales found under public/locales/*. Please run `npm run i18n:extract` or add your locale files.');
  }

  const missing = [];
  for (const k of used) {
    // require key to exist in default lang (en) if present, otherwise in any lang
    if (localeMap['en']) {
      if (!localeMap['en'].has(k)) missing.push({ key: k, reason: 'missing in en' });
    } else {
      const present = langs.some((l) => localeMap[l].has(k));
      if (!present) missing.push({ key: k, reason: 'missing in all locales' });
    }
  }

  const unused = [];
  for (const lang of Object.keys(localeMap)) {
    for (const k of localeMap[lang]) {
      if (!used.has(k)) unused.push({ key: k, lang });
    }
  }

  let failed = false;
  if (missing.length) {
    console.error('\nMissing translation keys (referenced in code but not found in locale JSON):');
    for (const m of missing) console.error(`  - ${m.key} (${m.reason})`);
    failed = true;
  }

  if (unused.length) {
    console.warn('\nUnused translation keys (present in locale files but not referenced in code):');
    for (const u of unused.slice(0, 50)) console.warn(`  - [${u.lang}] ${u.key}`);
    if (unused.length > 50) console.warn(`  ...and ${unused.length - 50} more`);
  }

  if (failed) {
    console.error('\ni18n check failed. Fix missing keys or update locale files.');
    process.exitCode = 1;
  } else {
    console.log('\ni18n check passed.');
  }
}

run();
