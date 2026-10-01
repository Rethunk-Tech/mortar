import { expect, test } from 'bun:test'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'
import {
  formatSize,
  meter,
  SUGGEST_FILE_AT,
  shownInfo,
  shownPreview,
  suggestFile,
  summarize,
} from './logic.ts'

const mod = (key: string, state: string, sizeKb: number, site = 'nexus'): Mod => ({
  key,
  site,
  name: key,
  author: '',
  version: '',
  state,
  reason: '',
  modId: 0,
  fileId: 0,
  repo: '',
  tag: '',
  asset: '',
  pageUrl: '',
  sizeKb,
  different: false,
  unverified: false,
  uniqueIds: [],
  enabled: true,
})

test('the meter fills up to the limit and warns before it', () => {
  expect(meter(498, 2000)).toEqual({ ratio: 0.249, level: 'ok' })
  expect(meter(1600, 2000).level).toBe('ok')
  expect(meter(1630, 2000).level).toBe('warn')
  expect(meter(2000, 2000)).toEqual({ ratio: 1, level: 'warn' })
  expect(meter(2400, 2000)).toEqual({ ratio: 1, level: 'over' })
  expect(meter(10, 0).ratio).toBe(0)
})

test('a file is suggested past about 240 mods or when the link cannot be made', () => {
  expect(suggestFile(SUGGEST_FILE_AT, false)).toBe(false)
  expect(suggestFile(SUGGEST_FILE_AT + 1, false)).toBe(true)
  expect(suggestFile(3, true)).toBe(true)
})

test('summary counts each state, sums only what downloads and skips unticked mods', () => {
  const mods = [
    mod('a', 'installed', 500),
    mod('b', 'download', 2048),
    mod('c', 'download', 1024, 'github'),
    mod('d', 'dependency', 512),
    mod('e', 'later', 0),
    mod('f', 'unavailable', 0),
    mod('g', 'download', 4096),
    mod('h', 'unknown state', 9),
  ]
  const s = summarize(mods, new Set(['g']))
  expect(s.counts).toEqual({ installed: 1, download: 2, dependency: 1, later: 1, unavailable: 1 })
  expect(s.leftOut).toBe(1)
  expect(s.toImport).toBe(4)
  expect(s.fromNexus).toBe(3)
  expect(s.sizeKb).toBe(2048 + 1024 + 512)
  // An unavailable mod is never importable, so unticking it changes nothing.
  expect(summarize(mods, new Set(['f'])).counts.unavailable).toBe(1)
})

test('sizes read as approximate KB, MB and GB', () => {
  expect(formatSize(0)).toBe('0 KB')
  expect(formatSize(900)).toBe('900 KB')
  expect(formatSize(1536)).toBe('1.5 MB')
  expect(formatSize(20 * 1024)).toBe('20 MB')
  expect(formatSize(1.5 * 1024 * 1024)).toBe('1.5 GB')
})

// Go writes a nil slice as null, whatever the generated types say.
test('null lists from Go become empty lists', () => {
  const info = shownInfo(JSON.parse('{"groups":[{"source":"nexus","mods":null}],"leftOut":null}'))
  expect(info.groups).toEqual([{ source: 'nexus', mods: [] }])
  expect(info.leftOut).toEqual([])
  const preview = shownPreview(JSON.parse('{"name":"x","mods":null,"problems":null}'))
  expect(preview.mods).toEqual([])
  expect(preview.problems).toEqual([])
})
