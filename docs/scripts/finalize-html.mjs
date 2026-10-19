// ZueDocs currently includes Google Fonts for its default typography. Noria
// uses three locally hosted fonts, so strip those unused network dependencies
// from built pages without forking ZueDocs or running client-side cleanup.
import { readdir, readFile, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export function removeUnusedFonts(html) {
  return html.replace(/<link\b[^>]*>/gi, (tag) =>
    /https:\/\/fonts\.(?:googleapis|gstatic)\.com\b/.test(tag) &&
    /rel=["'](?:preconnect|stylesheet)["']/.test(tag)
      ? '' : tag);
}

async function transformDirectory(folder) {
  for (const entry of await readdir(folder, { withFileTypes: true })) {
    const full = path.join(folder, entry.name);
    if (entry.isDirectory()) {
      await transformDirectory(full);
    } else if (entry.isFile() && entry.name.endsWith('.html')) {
      const original = await readFile(full, 'utf8');
      const optimized = removeUnusedFonts(original);
      if (original !== optimized) await writeFile(full, optimized);
    }
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const dist = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', 'dist');
  await transformDirectory(dist);
}
