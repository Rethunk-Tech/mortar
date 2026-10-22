// Fails when two messages in the source say the same thing: they differ only in case, punctuation or placeholder
// names. "…" and "?" stay significant, since a control that opens a dialog and the dialog's title are two messages.
// Also fails on a contraction: messages spell out "cannot" and "do not", so translators see one form. And on a
// typographic variant (curly quotes, dashes, arrows): messages use plain quotes and words, keeping only "…" and the
// "·" and "›" separators.
// The messages are extracted from the source into a scratch catalog, so a committed catalog that is behind cannot hide
// one.
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'

const FRONTEND = resolve(import.meta.dir, '../frontend')

function extractedCatalog(): string {
  const dir = mkdtempSync(join(tmpdir(), 'mortar-i18n-'))
  try {
    const config = join(dir, 'lingui.config.mjs')
    writeFileSync(
      config,
      `export default ${JSON.stringify({
        rootDir: FRONTEND,
        sourceLocale: 'en',
        locales: ['en'],
        catalogs: [{ path: join(dir, '{locale}', 'messages'), include: [join(FRONTEND, 'src')] }],
      })}\n`,
    )
    const run = Bun.spawnSync(
      [join(FRONTEND, 'node_modules/.bin/lingui'), 'extract', '--config', config],
      {
        cwd: FRONTEND,
        stdout: 'ignore',
        stderr: 'inherit',
      },
    )
    if (!run.success) {
      throw new Error('i18n: lingui extract failed')
    }
    return readFileSync(join(dir, 'en', 'messages.po'), 'utf8')
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
}

function msgids(po: string): string[] {
  const ids: string[] = []
  for (const block of po.split('\n\n')) {
    const match = /^msgid ((?:".*"\n?)+)/m.exec(block)
    const id = match?.[1] ? [...match[1].matchAll(/"(.*)"/g)].map((m) => m[1]).join('') : ''
    if (id !== '') {
      ids.push(id.replaceAll('\\"', '"'))
    }
  }
  return ids
}

const MIN_LETTERS = 4

function normalise(id: string): string | null {
  const flat = id
    .replace(/\{[\w.]+(?=, (?:plural|select|selectordinal),)/g, '{')
    .replace(/\{[\w.]+\}/g, '{}')
    .toLowerCase()
  // "{0} · {1}" and "{0} of {1}" are layout, not wording: nothing to say twice.
  if (flat.replace(/\{\}/g, '').replace(/[^\p{L}]/gu, '').length < MIN_LETTERS) {
    return null
  }
  return flat
    .replace(/[.,:;!'"‘’“”—–·()]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

function duplicates(ids: string[]): string[][] {
  const groups = new Map<string, string[]>()
  for (const id of ids) {
    const key = normalise(id)
    if (key !== null) {
      groups.set(key, [...(groups.get(key) ?? []), id])
    }
  }
  return [...groups.values()].filter((g) => g.length > 1)
}

const CONTRACTION =
  /\b\w+n['’]t\b|\b\w+['’](?:re|ve|ll|m|d)\b|\b(?:it|that|there|here|what|who|let)['’]s\b/i

function contractions(ids: string[]): string[] {
  return ids.filter((id) => CONTRACTION.test(id))
}

const VARIANT = /[“”‘’—–→←]/

function variants(ids: string[]): string[] {
  return ids.filter((id) => VARIANT.test(id))
}

if (import.meta.main) {
  const ids = msgids(extractedCatalog())
  const contracted = contractions(ids)
  for (const id of contracted) {
    console.error(
      `i18n: contraction in ${JSON.stringify(id)}; spell it out (docs/gui-design.md, Copy)`,
    )
  }
  const variant = variants(ids)
  for (const id of variant) {
    console.error(
      `i18n: typographic variant in ${JSON.stringify(id)}; use plain characters (docs/gui-design.md, Copy)`,
    )
  }
  const dupes = duplicates(ids)
  for (const group of dupes) {
    console.error(
      `i18n: one message written ${group.length} ways: ${group.map((g) => JSON.stringify(g)).join(' | ')}`,
    )
  }
  if (dupes.length > 0) {
    console.error(
      `i18n: ${dupes.length} duplicate messages; use one wording (docs/gui-design.md, Copy)`,
    )
  }
  if (dupes.length > 0 || contracted.length > 0 || variant.length > 0) {
    process.exit(1)
  }
}

export { contractions, duplicates, variants }
