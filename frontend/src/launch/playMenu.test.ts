import { expect, test } from 'bun:test'
import { playMenuEntries } from './playMenu.ts'

const preset = (key: string, isDefault = false) => ({ key, name: key, isDefault, base: false })

test('menu without presets is just Play without mods', () => {
  expect(playMenuEntries([])).toEqual([{ kind: 'vanilla' }])
})

test('focus order is each preset, then Play without mods, then Set default', () => {
  const kinds = playMenuEntries([preset('Standard', true), preset('a'), preset('b')]).map((e) =>
    e.kind === 'play' ? e.key : e.kind,
  )
  expect(kinds).toEqual(['Standard', 'a', 'b', 'vanilla', 'setDefault'])
})
