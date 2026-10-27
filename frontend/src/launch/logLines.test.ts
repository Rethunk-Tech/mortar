import { expect, test } from 'bun:test'
import {
  type Entry,
  Level,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { launchLogLines } from './logLines.ts'

const line = (seq: number, level: Level, load = false) => ({ seq, level, load }) as Entry

test('the log box leaves out per-mod loading and debug lines, and keeps the newest few', () => {
  const entries = [
    line(1, Level.Info),
    line(2, Level.Info, true),
    line(3, Level.Debug),
    line(4, Level.Trace),
    line(5, Level.Warn),
    line(6, Level.Error),
  ]
  expect(launchLogLines(entries, 5).map((e) => e.seq)).toEqual([1, 5, 6])
  expect(launchLogLines(entries, 2).map((e) => e.seq)).toEqual([5, 6])
})
