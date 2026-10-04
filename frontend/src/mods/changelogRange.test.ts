import { expect, test } from 'bun:test'
import type { Changelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import {
  changelogNoteIsRisky,
  changelogsBetween,
  changelogsHaveRiskyNotes,
} from './changelogRange.ts'

const log = (version: string): Changelog => ({ version, notes: [`notes ${version}`] })

test('keeps changelog versions newer than installed, including latest, newest first', () => {
  const logs = ['1.3.0', '1.2.0', '1.1.0', '1.0.0', '0.9.0'].map(log)
  expect(changelogsBetween(logs, '1.0.0', '1.3.0').map((c) => c.version)).toEqual([
    '1.3.0',
    '1.2.0',
    '1.1.0',
  ])
})

test('treats a leading v and pre-releases the way isNewer does', () => {
  const logs = ['1.1.0', '1.1.0-beta', '1.0.0'].map(log)
  expect(changelogsBetween(logs, '1.0.0', '1.1.0').map((c) => c.version)).toEqual([
    '1.1.0',
    '1.1.0-beta',
  ])
})

test('an empty or missing changelog list is empty', () => {
  expect(changelogsBetween(null, '1.0.0', '2.0.0')).toEqual([])
  expect(changelogsBetween([], '1.0.0', '2.0.0')).toEqual([])
})

test('changelogNoteIsRisky matches breaking-change and requirement phrases case-insensitively', () => {
  expect(changelogNoteIsRisky('BREAKING: rewrote saves')).toBe(true)
  expect(changelogNoteIsRisky('Save incompatible with 1.x')).toBe(true)
  expect(changelogNoteIsRisky('Now requires Content Patcher')).toBe(true)
  expect(changelogNoteIsRisky('You now needs a new dependency')).toBe(true)
  expect(changelogNoteIsRisky('Please start a new save file')).toBe(true)
  expect(changelogNoteIsRisky('Not save compatible with old farms')).toBe(true)
  expect(changelogNoteIsRisky('Remove before updating SMAPI')).toBe(true)
})

test('changelogNoteIsRisky ignores ordinary release notes', () => {
  expect(changelogNoteIsRisky('Fixed typo in dialog')).toBe(false)
  expect(changelogNoteIsRisky('Performance improvements')).toBe(false)
  expect(changelogNoteIsRisky('Updated for Stardew 1.6')).toBe(false)
})

test('changelogsHaveRiskyNotes is true when any note in the range matches', () => {
  const logs: Changelog[] = [
    { version: '2.0.0', notes: ['Bug fixes'] },
    { version: '1.1.0', notes: ['Breaking API change'] },
  ]
  expect(changelogsHaveRiskyNotes(logs)).toBe(true)
  expect(changelogsHaveRiskyNotes([{ version: '1.0.1', notes: ['Tweaks'] }])).toBe(false)
})
