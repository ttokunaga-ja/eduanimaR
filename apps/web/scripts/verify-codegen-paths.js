const { readFileSync } = require('node:fs');
const { execSync } = require('node:child_process');

function grep(pattern, target) {
  try {
    return execSync(`grep -RInE "${pattern}" ${target}`, {
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    }).trim();
  } catch (error) {
    if (error.status === 1) {
      return '';
    }
    throw error;
  }
}

const stalePathHits = grep('src/shared/api/generated|openapi/openapi\\.yaml', 'docs src');
if (stalePathHits) {
  console.error('Found stale codegen paths:');
  console.error(stalePathHits);
  process.exit(1);
}

const deepGenImportHits = grep('\\.\\./\\.\\./\\.\\./gen/openapi/web', 'src');
if (deepGenImportHits) {
  console.error('Found non-canonical deep imports to generated code:');
  console.error(deepGenImportHits);
  process.exit(1);
}

const indexSource = readFileSync('src/shared/api/index.ts', 'utf8');
if (!indexSource.includes("from '@gen/openapi/web/generated/")) {
  console.error("src/shared/api/index.ts must re-export generated clients via '@gen/openapi/web/generated/*'");
  process.exit(1);
}

console.log('✅ web codegen path checks passed');
