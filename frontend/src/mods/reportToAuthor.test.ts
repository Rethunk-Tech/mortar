import { expect, test } from 'bun:test'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { i18n } from '../i18n/index.ts'
import {
  buildModReport,
  buildModReportText,
  githubIssueURL,
  modErrorLines,
  truncateGithubBody,
} from './reportToAuthor.ts'

test('mod error lines keep continuations and cap at twenty', () => {
  const entries: Entry[] = [
    {
      seq: 1,
      time: '12:00:00',
      level: Level.Error,
      mod: 'SomeMod',
      message: 'first',
      cont: false,
    },
    {
      seq: 2,
      time: '12:00:00',
      level: Level.Error,
      mod: 'SomeMod',
      message: 'second line',
      cont: true,
    },
    {
      seq: 3,
      time: '12:00:01',
      level: Level.Info,
      mod: 'SomeMod',
      message: 'ignored',
      cont: false,
    },
  ]
  expect(modErrorLines(entries, 'SomeMod')).toEqual([
    '[12:00:00 ERROR SomeMod] first',
    'second line',
  ])
})

test('github issue URL encodes title and truncates body', () => {
  const body = `${'x'.repeat(7000)}`
  const url = githubIssueURL('Pathoschild/ContentPatcher', 'Error in CP', body)
  expect(url.startsWith('https://github.com/Pathoschild/ContentPatcher/issues/new?')).toBe(true)
  expect(url).toContain('title=Error+in+CP')
  expect(decodeURIComponent(url).includes('x'.repeat(6001))).toBe(false)
})

test('report text lists versions and errors', () => {
  const text = buildModReportText(i18n, {
    modName: 'Content Patcher',
    modVersion: '2.0.0',
    gameName: 'Stardew Valley',
    gameVersion: '1.6.15',
    smapiVersion: '4.1.10',
    mortarVersion: '0.1.0',
    errorLines: ['[12:00 ERROR CP] boom'],
    logShareURL: 'https://smapi.io/log/abc',
    source: { kind: 'github', name: '', repo: 'Pathoschild/ContentPatcher' },
    nexusDomain: 'stardewvalley',
    githubRepo: '',
    nexusModId: 0,
    issueTitle: '',
  })
  expect(text).toContain('Mod: Content Patcher 2.0.0')
  expect(text).toContain('SMAPI: 4.1.10')
  expect(text).toContain('Log: https://smapi.io/log/abc')
})

test('buildModReport prefers GitHub prefill', () => {
  const res = buildModReport(i18n, {
    modName: 'CP',
    modVersion: '1',
    gameName: 'Stardew Valley',
    gameVersion: '',
    smapiVersion: '',
    mortarVersion: '',
    errorLines: ['err'],
    logShareURL: '',
    source: { kind: 'github', name: '', repo: 'Pathoschild/ContentPatcher' },
    nexusDomain: 'stardewvalley',
    githubRepo: 'Pathoschild/ContentPatcher',
    nexusModId: 0,
    issueTitle: 'Oops',
  })
  expect(res.github).toBe(true)
  expect(res.url).toContain('github.com/Pathoschild/ContentPatcher/issues/new')
  expect(truncateGithubBody('abc')).toBe('abc')
})
