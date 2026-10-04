import { expect, test } from 'bun:test'
import { playMenuEntries } from './playMenu.ts'

const preset = (key: string, isDefault = false) => ({ key, name: key, isDefault, base: false })

test('menu without presets is just Play without mods', () => {
  expect(playMenuEntries([])).toEqual([{ kind: 'vanilla' }])
})

test('focus order is each preset, then Set default, then Play without mods', () => {
  const kinds = playMenuEntries([preset('Standard', true), preset('a'), preset('b')]).map((e) =>
    e.kind === 'play' ? e.key : e.kind,
  )
  expect(kinds).toEqual(['Standard', 'a', 'b', 'setDefault', 'vanilla'])
})
