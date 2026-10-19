// A control-only SDK query: the generator never yields a user turn.
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';

const sdkRoot = join(process.argv[2], 'node_modules', '@anthropic-ai', 'claude-agent-sdk');
const metadata = JSON.parse(readFileSync(join(sdkRoot, 'package.json'), 'utf8'));
if (metadata.version !== '0.3.287') throw new Error('Pinned SDK version required');
const require = createRequire(join(process.argv[2], 'package.json'));
const { query } = await import(pathToFileURL(require.resolve('@anthropic-ai/claude-agent-sdk')).href);
if (typeof query !== 'function') throw new Error('SDK query unavailable');

let release;
const parked = new Promise(resolve => { release = resolve; });
const prompt = (async function* () { await parked; })();
let handle;
const watchdog = setTimeout(() => process.exit(2), 25000);
try {
  handle = query({ prompt, options: {
    settingSources: [], tools: [], persistSession: false,
    pathToClaudeCodeExecutable: process.argv[3],
    canUseTool: async () => ({ behavior: 'deny', message: 'Tools disabled for quota control.' }),
  }});
  const method = handle?.usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET;
  if (typeof method !== 'function') throw new Error('SDK usage control unavailable');
  const result = await method.call(handle);
  if (!result || typeof result !== 'object' || typeof result.rate_limits_available !== 'boolean' ||
      !Object.hasOwn(result, 'rate_limits')) throw new Error('SDK usage schema changed');
  // Emit only quota fields. Never let session metadata or credentials escape.
  process.stdout.write(JSON.stringify({rate_limits_available: result.rate_limits_available, rate_limits: result.rate_limits}));
} catch {
  process.exitCode = 1;
} finally {
  release();
  await Promise.race([
    Promise.allSettled([
      (async () => { if (typeof handle?.interrupt === 'function') await handle.interrupt(); })(),
      (async () => { if (typeof handle?.close === 'function') await handle.close(); })(),
    ]),
    new Promise(resolve => setTimeout(resolve, 1000)),
  ]);
  clearTimeout(watchdog);
  process.exit(process.exitCode || 0);
}
