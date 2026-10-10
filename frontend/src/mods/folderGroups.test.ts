import { expect, test } from 'bun:test'
import type {
  Entry,
  Source,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { folderRows } from './folderGroups.ts'

const source = { kind: 'curseforge', name: 'Pack', fileId: 10 } as Source

function entry(over: Partial<Entry> & { key: string }): Entry {
  return { source, mods: [], disabled: [], previousKey: '', ...over } as Entry
}

function perFile(item: string, file: string, off = false): Entry {
  const key = `${item}#${file}`
  const id = `folder:${key}`
  return entry({
    key,
    item,
    file,
    mods: [{ id, name: file }] as Entry['mods'],
    disabled: off ? [id] : [],
  })
}

test('per-file entries of one archive form one group with the archive source and counts', () => {
  const rows = folderRows([
    perFile('pkg', 'a.package'),
    perFile('pkg', 'b.package', true),
    perFile('pkg', 'c.package'),
  ])
  expect(rows).toHaveLength(1)
  const g = rows[0]
  if (g?.kind !== 'archive') {
    throw new Error('expected a group')
  }
  expect(g.item).toBe('pkg')
  expect(g.source).toBe(source)
  expect(g.files).toBe(3)
  expect(g.off).toBe(1)
  expect(g.entries.map((e) => e.file)).toEqual(['a.package', 'b.package', 'c.package'])
})

test('the Tray entry joins its archive group, wherever it sits in the profile', () => {
  const tray = entry({ key: 'pkg#tray', item: 'pkg' })
  const rows = folderRows([tray, perFile('pkg', 'a.package')])
  expect(rows).toHaveLength(1)
  expect(rows[0]).toMatchObject({ kind: 'archive', files: 1, tray })
})

test('a kept-whole archive and a non-folder entry stay single rows, the whole archive keeping its Tray entry', () => {
  const whole = entry({ key: 'whole' })
  const tray = entry({ key: 'whole#tray', item: 'whole' })
  const plain = entry({ key: 'other' })
  const rows = folderRows([whole, plain, tray])
  expect(rows).toEqual([
    { kind: 'single', entry: whole, tray },
    { kind: 'single', entry: plain, tray: undefined },
  ])
})

test('an archive with only Tray files is a single row, and groups keep the profile order', () => {
  const trayOnly = entry({ key: 'h#tray', item: 'h' })
  const rows = folderRows([
    perFile('a', 'x.package'),
    trayOnly,
    perFile('b', 'y.package'),
    perFile('a', 'z.package'),
  ])
  expect(
    rows.map((r) =>
      r.kind === 'archive' ? `archive:${r.item}:${r.files}` : `single:${r.entry.key}`,
    ),
  ).toEqual(['archive:a:2', 'single:h#tray', 'archive:b:1'])
})
