import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const docs = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repo = path.dirname(docs);

test('Vercel Git deployments and ignored builds are disabled', () => {
  const config = JSON.parse(readFileSync(path.join(docs, 'vercel.json'), 'utf8'));
  assert.equal(config.git.deploymentEnabled, false);
  assert.equal(Object.hasOwn(config, 'ignoreCommand'), false);
  assert.equal(config.outputDirectory, 'dist');
});

test('only website inputs trigger production docs deployment', () => {
  const workflow = readFileSync(path.join(repo, '.github/workflows/docs-deploy.yml'), 'utf8');
  for (const event of ['push', 'pull_request']) {
    const section = workflow.match(new RegExp(`^  ${event}:\\n([\\s\\S]*?)(?=^  (?:push|pull_request|workflow_dispatch):|^permissions:)`, 'm'))?.[1];
    assert.ok(section, `missing ${event} trigger`);
    assert.match(section, /branches: \[main\]/);
    const paths = [...section.matchAll(/^      - '([^']+)'$/gm)].map(match => match[1]);
    assert.deepEqual(paths, ['docs/**', 'scripts/install.sh', 'scripts/install.ps1']);
  }
  assert.match(workflow, /^  workflow_dispatch:$/m);
  assert.match(workflow, /github\.ref == 'refs\/heads\/main'/);
  assert.match(workflow, /vercel@latest deploy --prebuilt --prod/);
  assert.match(workflow, /secrets\.VERCEL_TOKEN/);
  assert.match(workflow, /vars\.VERCEL_ORG_ID/);
  assert.match(workflow, /vars\.VERCEL_PROJECT_ID/);
  assert.match(workflow, /node scripts\/prepare-prebuilt\.mjs/);
  assert.match(workflow, /bun run build/);
});

test('the root installers and website installers are identical', () => {
  for (const filename of ['install.sh', 'install.ps1']) {
    assert.deepEqual(
      readFileSync(path.join(repo, 'scripts', filename)),
      readFileSync(path.join(docs, 'public', filename)),
    );
  }
});
