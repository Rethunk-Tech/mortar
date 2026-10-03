import { expect, test } from 'bun:test'
import { formatProblemReport, whyKeysOf } from './problemReport.ts'

test('report counts problems by section, lists Why? keys, and includes the harmless count', () => {
  expect(
    formatProblemReport(
      [
        { title: 'Missing requirements', count: 2 },
        {
          title: 'Conflicts',
          count: 1,
          whyKeys: whyKeysOf([{ target: 'Portraits/Abigail', packId: 'Pack.A' }]),
          lines: ['SVE overlaps Portraits/Abigail'],
        },
        { title: 'Cleanup', count: 1, lines: ['Content Patcher: Not needed by any enabled mod'] },
        { title: 'Dismissed', count: 1, lines: ['Old overlap'] },
      ],
      'Harmless',
      3,
    ),
  ).toBe(
    [
      'Missing requirements: 2',
      'Conflicts: 1',
      '  Portraits/Abigail',
      '  SVE overlaps Portraits/Abigail',
      'Cleanup: 1',
      '  Content Patcher: Not needed by any enabled mod',
      'Dismissed: 1',
      '  Old overlap',
      'Harmless: 3',
    ].join('\n'),
  )
})

test('Why? keys fall back to pack id when the target is empty', () => {
  expect(
    whyKeysOf([
      { packId: 'Pack.A', target: '' },
      { packId: '', target: '' },
    ]),
  ).toEqual(['Pack.A'])
})
