// The render budget's gallery: one page with every screen loading and settled, in Chromium and in WebKitGTK, beside its
// figures and layout breaks, for a person to look over after a run.
import { existsSync, readdirSync, writeFileSync } from 'node:fs'
import { join, relative } from 'node:path'
import type { Result } from './render-budget.ts'

const ENGINES = ['chromium', 'webkit'] as const

const esc = (s: string) =>
  s
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')

function picture(
  dirs: { out: string; shots: string },
  engine: string,
  phase: 'loading' | 'settled',
  name: string,
) {
  const { out, shots } = dirs
  const file = join(shots, engine, ...(phase === 'loading' ? ['loading'] : []), `${name}.png`)
  const label = `${engine === 'chromium' ? 'Chromium' : 'WebKitGTK'} · ${phase === 'loading' ? 'Loading' : 'Settled'}`
  if (!existsSync(file)) {
    return `<figure class="none"><figcaption>${label}</figcaption><div>${engine === 'webkit' ? 'WebKitGTK walks screens, not interactions' : 'not taken'}</div></figure>`
  }
  const src = esc(relative(out, file))
  return `<figure><figcaption>${label}</figcaption><a href="${src}"><img src="${src}" loading="lazy" alt="${esc(`${name}, ${label}`)}"></a></figure>`
}

const count = (n: number, noun: string): string => `${n} ${noun}${n === 1 ? '' : 's'}`

function figures(r: Result | undefined): string {
  if (!r) {
    return '<p class="meta">WebKitGTK only: no React figures</p>'
  }
  if (r.failure) {
    return `<p class="bad">failed: ${esc(r.failure)}</p>`
  }
  const a = r.action
  const parts = a
    ? [count(a.updates, 're-render'), count(a.calls, 'backend call'), `settled ${a.settledMs} ms`]
    : []
  return `<p class="meta">${esc(parts.join(' · '))}</p>${r.over.map((o) => `<p class="bad">over: ${esc(o)}</p>`).join('')}`
}

/** Writes <out>/index.html and returns its path. */
export function writeGallery(
  out: string,
  shots: string,
  results: Result[],
  breaks: string[],
): string {
  const byName = new Map(results.map((r) => [r.name, r]))
  const names = results.map((r) => r.name)
  const webkitDir = join(shots, 'webkit')
  for (const f of existsSync(webkitDir) ? readdirSync(webkitDir) : []) {
    const name = f.replace(/\.png$/, '')
    if (f.endsWith('.png') && !names.includes(name)) {
      names.push(name)
    }
  }
  const breaksOf = (name: string) => breaks.filter((b) => b.split(' | ')[0] === `LAYOUT ${name}`)
  const bad = (name: string) => {
    const r = byName.get(name)
    return Boolean(r?.failure) || (r?.over.length ?? 0) > 0 || breaksOf(name).length > 0
  }
  const ordered = [...names.filter(bad), ...names.filter((n) => !bad(n))]
  const toc = ordered
    .map((n) => `<li${bad(n) ? ' class="bad"' : ''}><a href="#${esc(n)}">${esc(n)}</a></li>`)
    .join('')
  const rows = names
    .map(
      (n) => `<section id="${esc(n)}"${bad(n) ? ' class="flag"' : ''}>
<h2>${esc(n)}</h2>${figures(byName.get(n))}${breaksOf(n)
        .map((b) => `<p class="bad">${esc(b.split(' | ').slice(1).join(' | '))}</p>`)
        .join('')}
<div class="shots">${ENGINES.flatMap((e) => [picture({ out, shots }, e, 'loading', n), picture({ out, shots }, e, 'settled', n)]).join('')}</div>
</section>`,
    )
    .join('\n')
  const flagged = names.filter(bad).length
  const page = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="darkreader-lock"><title>Render budget</title>
<style>
:root { color-scheme: dark; --bg: #16161a; --fg: #e8e6e3; --muted: #9a978f; --line: #333; --bad: #ff7a6b; }
body { margin: 0; background: var(--bg); color: var(--fg); font: 14px/1.4 system-ui, sans-serif; }
header { padding: 16px; border-bottom: 1px solid var(--line); }
header ul { columns: 4 200px; margin: 8px 0 0; padding-left: 18px; }
a { color: inherit; }
section { padding: 16px; border-bottom: 1px solid var(--line); }
section.flag { border-left: 4px solid var(--bad); }
h1, h2 { margin: 0 0 4px; }
h2 { font-size: 16px; }
.meta { color: var(--muted); margin: 0 0 8px; }
.bad { color: var(--bad); margin: 0 0 4px; }
.shots { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 12px; }
figure { margin: 0; }
figcaption { color: var(--muted); font-size: 12px; margin-bottom: 4px; }
img { width: 100%; height: auto; border: 1px solid var(--line); display: block; }
.none div { border: 1px dashed var(--line); color: var(--muted); padding: 24px; text-align: center; }
</style></head><body>
<header><h1>Render budget</h1><p class="meta">${names.length} screens, ${flagged} with a problem (listed first). Pictures open full size.</p><ul>${toc}</ul></header>
${rows}
</body></html>
`
  const file = join(out, 'index.html')
  writeFileSync(file, page)
  return file
}
