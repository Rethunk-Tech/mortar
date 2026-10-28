import { beforeEach, expect, test } from 'bun:test'
import type { Settings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { inFolder, targetOf } from './autoInstall.ts'

test('inFolder matches the folder itself and children under either separator', () => {
  expect(inFolder('/mods', '/mods')).toBe(true)
  expect(inFolder('/mods/a', '/mods')).toBe(true)
  expect(inFolder('C:\\Mods\\a', 'C:\\Mods')).toBe(true)
  expect(inFolder('C:\\Mods2', 'C:\\Mods')).toBe(false)
  expect(inFolder('/mods2/a', '/mods')).toBe(false)
})

const picked = (id: string) =>
  ({ games: { g: { extraModsProfiles: { '/mods': id } } } }) as unknown as Settings

beforeEach(() => {
  useSettings.setState({ games: null })
  useProfiles.setState({ game: null, openId: '', profiles: [] } as never)
})

test('targetOf prefers the picked profile', () => {
  useSettings.setState(picked('p1'))
  useProfiles.setState({ game: { id: 'g' }, openId: 'p2', profiles: [{ id: 'p1' }] } as never)
  expect(targetOf('g', '/mods')).toBe('p1')
})

test('targetOf gives nothing for a picked profile that was deleted', () => {
  useSettings.setState(picked('gone'))
  useProfiles.setState({ game: { id: 'g' }, openId: 'p2', profiles: [{ id: 'p2' }] } as never)
  expect(targetOf('g', '/mods')).toBe('')
})

test('targetOf falls back to the open profile of the open game only', () => {
  useProfiles.setState({ game: { id: 'g' }, openId: 'p2', profiles: [{ id: 'p2' }] } as never)
  expect(targetOf('g', '/mods')).toBe('p2')
  expect(targetOf('other', '/mods')).toBe('')
})
