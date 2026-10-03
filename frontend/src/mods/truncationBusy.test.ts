import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const src = (...parts: string[]) => readFileSync(join(import.meta.dir, ...parts), 'utf8')

test('mod cards expose a tooltip for clipped author/version and first tag', () => {
  const cards = src('ModCards.tsx')
  expect(cards).toContain('title={meta}')
  expect(cards).toContain('title={tag}')
})

test('also-in-profiles rows and load-order names truncate with a title', () => {
  expect(/title=\{`/.test(src('Sidebar.tsx'))).toBe(true)
  expect(/r\.profileName/.test(src('Sidebar.tsx'))).toBe(true)
  const load = src('LoadOrderTab.tsx')
  expect(load).toContain('noWrap={true}')
  expect(load).toContain('title={row.name')
})

test('saves copy-from uses a short label, tooltip, and usePending', () => {
  const saves = src('..', 'saves', 'lackChipActions.tsx')
  expect(saves).toContain('const [copying, runCopy] = usePending()')
  expect(saves).toContain('title={source.name}')
  expect(saves).toContain('disabled={copying || locked}')
  expect(saves).toContain('{t`Copy`}')
  expect(/Copy from \$\{source\.name\}/.test(saves)).toBe(false)
})

test('find-mod hits drop UniqueID from the visible label and keep it in the tooltip', () => {
  const page = src('..', 'profiles', 'ProfilesPage.tsx')
  expect(page).toContain('title={`')
  expect(page).toContain('h.uniqueId')
  expect(page).toContain('minWidth: 0')
  const kids = page.split('FindModSearch')[1] ?? ''
  expect(/title=\{`\$\{h\.name\} · \$\{h\.uniqueId\}/.test(kids)).toBe(true)
  expect(/\{`\$\{h\.name\} · \$\{h\.profileName\}/.test(kids)).toBe(true)
})

test('search runs and game-mods preview use LoadingRow while waiting', () => {
  expect(src('..', 'console', 'SearchRunsDialog.tsx')).toContain(
    '<LoadingRow>{t`Searching…`}</LoadingRow>',
  )
  const dialog = src('..', 'profiles', 'GameModsDialog.tsx')
  expect(dialog).toContain('previewing ? (')
  expect(dialog).toContain('<LoadingRow>')
})

test('queue profile line has a title and retry is guarded', () => {
  expect(src('..', 'queue', 'Callout.tsx')).toContain('title={profile}')
  const queue = src('..', 'queue', 'QueueBody.tsx')
  expect(queue).toContain('onClick={() => run(() => RetryFailed())}')
})
