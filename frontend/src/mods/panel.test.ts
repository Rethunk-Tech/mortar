import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { showModId, useDetail } from './detail.ts'
import { modId } from './lookup.ts'
import { clampPanelWidth } from './panelWidth.ts'
import { useMods } from './store.ts'

const mod = (id: string) => ({ key: 'k', id, name: id }) as Mod

test('moving the selection by id shows that mod and clears when none matches', () => {
  useMods.setState({ mods: [mod('a'), mod('b')] })
  showModId(modId(mod('b')))
  expect(useDetail.getState().detailId).toBe('k/b')
  showModId('k/missing')
  expect(useDetail.getState().detailId).toBe('')
})

test('the panel keeps between 300 px and half the Mods tab', () => {
  expect(clampPanelWidth(100, 1200)).toBe(300)
  expect(clampPanelWidth(900, 1200)).toBe(600)
  expect(clampPanelWidth(400, 1200)).toBe(400)
})
