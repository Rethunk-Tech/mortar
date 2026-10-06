import { expect, test } from 'bun:test'
import { build } from './changelog.ts'

test('groups feat, fix and perf by area and hides the rest', () => {
  const out = build([
    'feat(browse): the merged browse hides installed mods.',
    'fix(profile): a rename keeps its group',
    'perf(launch): read the mods once',
    'feat(unknownscope): something new',
    'refactor(profile): split a file',
    'test(lan): cover pairing',
    'chore(i18n): extract',
    'ci: pin an action',
    'style(queue): group imports',
    'feat(queuesvc): internal scope',
    'fix(browse): handles `rawField` names',
    'not a conventional commit',
  ])
  expect(out).toBe(
    [
      '## Mods and browsing\n\n- The merged browse hides installed mods\n',
      '## Profiles and sharing\n\n- Fixed: A rename keeps its group\n',
      '## Launching and loaders\n\n- Read the mods once\n',
      '## Other\n\n- Something new\n',
    ].join('\n'),
  )
})

test('an empty range says so', () => {
  expect(build(['chore(deps): bump'])).toBe('No user-facing changes.\n')
})
