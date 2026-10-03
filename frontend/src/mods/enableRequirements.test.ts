import { expect, test } from 'bun:test'
import { enableRequirementsDecision, pendingRequired } from './enableRequirements.ts'

const mod = (
  uniqueId: string,
  over: {
    name?: string
    enabled?: boolean
    needs?: string[]
    optional?: string[]
    contentPackFor?: string
  } = {},
) => ({
  uniqueId,
  name: over.name ?? uniqueId,
  enabled: over.enabled ?? true,
  needs: over.needs ?? [],
  optional: over.optional ?? [],
  ...(over.contentPackFor === undefined ? {} : { contentPackFor: over.contentPackFor }),
})

test('pendingRequired lists disabled required deps already in the profile', () => {
  const core = mod('Me.Core', { name: 'Core', enabled: false })
  const user = mod('Me.User', { name: 'User', needs: ['Me.Core'] })
  expect(pendingRequired([core, user], [user])).toEqual([core])
})

test('pendingRequired includes uninstalled-but-present (disabled) transitive deps', () => {
  const base = mod('Me.Base', { name: 'Base', enabled: false })
  const mid = mod('Me.Mid', { name: 'Mid', enabled: false, needs: ['Me.Base'] })
  const user = mod('Me.User', { needs: ['Me.Mid'] })
  expect(pendingRequired([base, mid, user], [user]).map((m) => m.uniqueId)).toEqual([
    'Me.Mid',
    'Me.Base',
  ])
})

test('pendingRequired skips optional needs and already-enabled mods', () => {
  const opt = mod('Me.Opt', { enabled: false })
  const on = mod('Me.On', { enabled: true })
  const user = mod('Me.User', { needs: ['Me.Opt', 'Me.On'], optional: ['Me.Opt'] })
  expect(pendingRequired([opt, on, user], [user])).toEqual([])
})

test('pendingRequired skips required mods that are not in the profile', () => {
  const user = mod('Me.User', { needs: ['Gone.Mod'] })
  expect(pendingRequired([user], [user])).toEqual([])
})

test('enableRequirementsDecision is enable, ask, or skip', () => {
  expect(enableRequirementsDecision('always', 2)).toBe('enable')
  expect(enableRequirementsDecision('', 1)).toBe('enable')
  expect(enableRequirementsDecision('ask', 1)).toBe('ask')
  expect(enableRequirementsDecision('never', 3)).toBe('skip')
  expect(enableRequirementsDecision('ask', 0)).toBe('skip')
  expect(enableRequirementsDecision('always', 0)).toBe('skip')
})
