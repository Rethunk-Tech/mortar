import { expect, test } from 'bun:test'
import { reportInput } from './reportToAuthor.ts'

const parts = {
  game: 'stardewvalley',
  modName: 'Content Patcher',
  modVersion: '2.0.0',
  modLogName: 'Content Patcher',
  source: { kind: 'github', name: '', repo: 'Pathoschild/ContentPatcher' },
  loader: { gameVersion: '1.6.15', version: '4.1.10' },
  loaderName: 'SMAPI',
  mortarVersion: '0.1.0',
  logShareURL: 'https://smapi.io/log/abc',
  issueTitle: 'Oops',
  target: { repo: 'other/repo', url: '' },
}

test('a GitHub source reports to its own repo, not the site target', () => {
  const input = reportInput(parts)
  expect(input).toMatchObject({
    Game: 'stardewvalley',
    GameVersion: '1.6.15',
    SMAPIVersion: '4.1.10',
    LoaderName: 'SMAPI',
    LogName: 'Content Patcher',
    GitHubRepo: '',
    IssueTitle: 'Oops',
  })
  expect(input.Source.repo).toBe('Pathoschild/ContentPatcher')
})

test('a site mod without a source takes the site target and an empty loader', () => {
  const input = reportInput({
    ...parts,
    source: undefined,
    loader: null,
    target: { repo: 'a/b', url: 'https://mods.vintagestory.at/carryon' },
  })
  expect(input).toMatchObject({
    GitHubRepo: 'a/b',
    PageURL: 'https://mods.vintagestory.at/carryon',
    GameVersion: '',
    SMAPIVersion: '',
  })
})
