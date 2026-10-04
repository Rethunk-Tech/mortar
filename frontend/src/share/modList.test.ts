import { expect, test } from 'bun:test'
import type {
  Entry,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  DISCORD_LIMIT,
  escapeMarkdown,
  formatDiscord,
  formatMarkdown,
  formatPlain,
  listItems,
  type ModListItem,
  splitDiscord,
} from './modList.ts'

const item = (
  name: string,
  version: string,
  url = 'https://www.nexusmods.com/stardewvalley/mods/1',
  enabled = true,
): ModListItem => ({ name, version, url, enabled })

const source = (kind: string, extra: Entry['source'] = { kind, name: kind }): Entry['source'] =>
  extra

const entry = (partial: Partial<Entry> & Pick<Entry, 'key' | 'source'>): Entry => ({
  previousKey: '',
  mods: [],
  disabled: [],
  ...partial,
})

const profile = (entries: Entry[]): Profile => ({
  id: 'p1',
  name: 'Farm',
  notes: '',
  cover: '',
  order: 0,
  hidden: false,
  created: '',
  updated: '',
  entries,
})

test('listItems drops bundled SMAPI and the bridge and keeps disabled mods', () => {
  const items = listItems(
    profile([
      entry({
        key: 'smapi-1',
        source: source('smapi', { kind: 'smapi', name: 'SMAPI' }),
        mods: [
          {
            uniqueId: 'SMAPI.ConsoleCommands',
            version: '1',
            name: 'Console',
            author: '',
            folder: '.',
          },
        ],
      }),
      entry({
        key: 'bridge-1',
        source: source('mortar', { kind: 'mortar', name: 'Mortar' }),
        mods: [
          { uniqueId: 'Mortar.Bridge', version: '1', name: 'Bridge', author: '', folder: '.' },
        ],
      }),
      entry({
        key: 'content-1',
        source: source('nexus', { kind: 'nexus', name: 'CP', modId: 1915 }),
        mods: [
          {
            uniqueId: 'Pathoschild.ContentPatcher',
            version: '2.0.0',
            name: 'Content Patcher',
            author: '',
            folder: '.',
          },
        ],
        disabled: ['Pathoschild.ContentPatcher'],
      }),
    ]),
  )
  expect(items).toEqual([
    {
      name: 'Content Patcher',
      version: '2.0.0',
      url: 'https://www.nexusmods.com/stardewvalley/mods/1915',
      enabled: false,
    },
  ])
})

test('listItems can keep one entry and builds a GitHub page URL', () => {
  const items = listItems(
    profile([
      entry({
        key: 'keep',
        source: source('github', { kind: 'github', name: 'asset', repo: 'owner/mod' }),
        mods: [{ uniqueId: 'A.Mod', version: '3', name: 'A Mod', author: '', folder: '.' }],
      }),
      entry({
        key: 'skip',
        source: source('nexus', { kind: 'nexus', name: 'Other', modId: 2 }),
        mods: [{ uniqueId: 'B.Mod', version: '1', name: 'B', author: '', folder: '.' }],
      }),
    ]),
    ['keep'],
  )
  expect(items).toEqual([
    { name: 'A Mod', version: '3', url: 'https://github.com/owner/mod', enabled: true },
  ])
})

test('Markdown is a bulleted list of linked names and versions', () => {
  expect(
    formatMarkdown([
      item('Content Patcher', '2.8.3', 'https://www.nexusmods.com/stardewvalley/mods/1915'),
      item('Json Assets', '1.2', 'https://github.com/spacechase0/JsonAssets'),
    ]),
  ).toBe(
    [
      '- [Content Patcher](https://www.nexusmods.com/stardewvalley/mods/1915) 2.8.3',
      '- [Json Assets](https://github.com/spacechase0/JsonAssets) 1.2',
    ].join('\n'),
  )
})

test('Markdown and plain text group by enabled when any mod is off', () => {
  const items = [
    item('On', '1', 'https://example.com/on', true),
    item('Off', '2', 'https://example.com/off', false),
  ]
  expect(formatMarkdown(items)).toBe(
    [
      'On',
      '- [On](https://example.com/on) 1',
      '',
      'Switched off',
      '- [Off](https://example.com/off) 2',
    ].join('\n'),
  )
  expect(formatPlain(items)).toBe(
    [
      'On',
      'On v1 - https://example.com/on',
      '',
      'Switched off',
      'Off v2 - https://example.com/off',
    ].join('\n'),
  )
})

test('plain text is Name vVersion - URL', () => {
  expect(
    formatPlain([
      item('Content Patcher', '2.8.3', 'https://www.nexusmods.com/stardewvalley/mods/1915'),
    ]),
  ).toBe('Content Patcher v2.8.3 - https://www.nexusmods.com/stardewvalley/mods/1915')
})

test('names with Markdown special characters are escaped', () => {
  const name = 'Mod *[beta]* (new)_x'
  expect(escapeMarkdown(name)).toBe('Mod \\*\\[beta\\]\\* \\(new\\)\\_x')
  expect(formatMarkdown([item(name, '1.0', 'https://example.com/m')])).toBe(
    '- [Mod \\*\\[beta\\]\\* \\(new\\)\\_x](https://example.com/m) 1.0',
  )
})

test('Discord keeps Markdown under 2000 characters per message', () => {
  const items = Array.from({ length: 80 }, (_, i) =>
    item(
      `Mod ${String(i).padStart(2, '0')}`,
      '1.0.0',
      'https://www.nexusmods.com/stardewvalley/mods/1915',
    ),
  )
  const all = formatMarkdown(items)
  expect(all.length).toBeGreaterThan(DISCORD_LIMIT)
  const parts = formatDiscord(items)
  expect(parts.length).toBeGreaterThan(1)
  for (const part of parts) {
    expect(part.length).toBeLessThanOrEqual(DISCORD_LIMIT)
  }
  expect(parts.join('\n')).toBe(all)
})

test('a single overlong line is sliced at the Discord limit', () => {
  const line = 'a'.repeat(DISCORD_LIMIT + 50)
  const parts = splitDiscord(line)
  expect(parts).toEqual(['a'.repeat(DISCORD_LIMIT), 'a'.repeat(50)])
})
