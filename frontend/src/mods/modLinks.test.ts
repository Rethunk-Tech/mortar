import { expect, test } from 'bun:test'
import { modLinksOf } from './modLinks.ts'
import type { Row } from './problemGroups.ts'

const row = (r: unknown) => r as Row
const m = (n: string) => ({ name: n, key: `k${n}`, id: `i${n}` })

test('a broken mod links its cycle, else itself', () => {
  expect(
    modLinksOf(row({ kind: 'broken', broken: { ...m('a'), cycle: [m('b'), m('c')] } })),
  ).toEqual([m('b'), m('c')])
  expect(modLinksOf(row({ kind: 'broken', broken: { ...m('a'), cycle: [] } }))).toEqual([m('a')])
  expect(modLinksOf(row({ kind: 'broken', broken: m('a') }))).toEqual([m('a')])
})

test('single-mod kinds give one link', () => {
  expect(modLinksOf(row({ kind: 'runError', runError: m('a') }))).toEqual([m('a')])
  expect(modLinksOf(row({ kind: 'setting', setting: m('a') }))).toEqual([m('a')])
  expect(modLinksOf(row({ kind: 'deprecated', deprecated: m('a') }))).toEqual([m('a')])
  expect(modLinksOf(row({ kind: 'loadFailure', loadFailure: m('a') }))).toEqual([m('a')])
})

test('a load failure with no key and a duplicate with no copies link nothing', () => {
  expect(modLinksOf(row({ kind: 'loadFailure', loadFailure: { ...m('a'), key: '' } }))).toEqual([])
  expect(modLinksOf(row({ kind: 'duplicate', duplicate: { ...m('a'), copies: [] } }))).toEqual([])
  expect(modLinksOf(row({ kind: 'duplicate', duplicate: { ...m('a'), copies: null } }))).toEqual([])
  expect(
    modLinksOf(row({ kind: 'duplicate', duplicate: { ...m('a'), copies: [{ key: 'c1' }] } })),
  ).toEqual([{ name: 'a', key: 'c1', id: 'ia' }])
})

test('asset links drop names without a matching key or pack id', () => {
  expect(
    modLinksOf(
      row({
        kind: 'asset',
        asset: { names: ['x', 'y', 'z'], keys: ['kx', 'ky'], packIds: ['px'] },
      }),
    ),
  ).toEqual([{ name: 'x', key: 'kx', id: 'px' }])
})

test('a plugin clash links each copy; other kinds link nothing', () => {
  expect(
    modLinksOf(row({ kind: 'pluginClash', pluginClash: { copies: [m('a'), m('b')] } })),
  ).toEqual([m('a'), m('b')])
  expect(modLinksOf(row({ kind: 'pluginClash', pluginClash: {} }))).toEqual([])
  expect(modLinksOf(row({ kind: 'damaged', damaged: m('a') }))).toEqual([])
})
