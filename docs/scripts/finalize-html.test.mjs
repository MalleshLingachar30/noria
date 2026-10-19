import test from 'node:test';
import assert from 'node:assert/strict';
import { removeUnusedFonts } from './finalize-html.mjs';

test('remove only unused ZueDocs default font resources', () => {
  const html = `<head>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400&display=swap" rel="stylesheet">
    <link href="/fonts/bricolage-grotesque-latin.woff2" rel="preload" as="font">
    <link href="/favicon.svg" rel="icon">
  </head>`;
  const transformed = removeUnusedFonts(html);
  assert.ok(!transformed.includes('fonts.googleapis.com'));
  assert.ok(!transformed.includes('fonts.gstatic.com'));
  assert.ok(transformed.includes('/fonts/bricolage-grotesque-latin.woff2'));
  assert.ok(transformed.includes('/favicon.svg'));
  assert.equal(removeUnusedFonts(transformed), transformed);
});
