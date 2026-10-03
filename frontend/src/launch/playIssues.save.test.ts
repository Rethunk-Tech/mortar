import { expect, test } from 'bun:test'
import { playIssueSummary } from './playIssues.ts'

test('playIssueSummary lists recorded mods this profile lacks', () => {
  expect(
    playIssueSummary({
      currentProfileId: 'p1',
      saveMods: [{ name: 'Alpha' }, { name: 'Beta' }],
      switchProfileId: 'p2',
    }),
  ).toEqual([
    {
      kind: 'saveMods',
      count: 2,
      names: ['Alpha', 'Beta'],
      switchProfileId: 'p2',
    },
  ])
})
