import { expect, test } from 'bun:test'
import { draftIssue, firstDraft } from './draft.ts'

const ready = {
  ...firstDraft({ author: 'Ab', team: 'Ab', guidPrefix: 'com.ab', parent: '/mods' }, ''),
  name: 'ShopDrop',
  description: 'Lands somewhere new.',
}

test('a remembered author, team, prefix and folder leave only the name and description to fill', () => {
  expect(
    draftIssue(firstDraft({ author: 'Ab', team: 'Ab', guidPrefix: 'com.ab', parent: '/mods' }, '')),
  ).not.toBe('')
  expect(draftIssue(ready)).toBe('')
})

test('names Mortar would refuse are caught before it is asked', () => {
  expect(draftIssue({ ...ready, name: 'shop drop' })).not.toBe('')
  expect(draftIssue({ ...ready, repo: '-x' })).not.toBe('')
  expect(draftIssue({ ...ready, github: true, owner: '--help' })).not.toBe('')
  expect(draftIssue({ ...ready, github: true, owner: 'albinogeek' })).toBe('')
})
