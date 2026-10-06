import { expect, mock, test } from 'bun:test'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: unknown[]) => String.raw({ raw: parts }, ...values),
  plural: (n: number, forms: { other: string }) => forms.other.replace('#', String(n)),
}))

const { offlineMessage } = await import('./offlineText.ts')

const NEVER = '0001-01-01T00:00:00Z'
const down = (id: string, lastOK: string) => ({
  id,
  unreachable: true,
  lastOK,
  lastFail: NEVER,
  lastError: 'dial tcp: refused',
  lastReason: 'refused',
})

test('the offline sentence gives each down source its own saved time, or says nothing is saved', () => {
  const both = offlineMessage([down('github', '2026-10-05T12:00:00Z'), down('nexus', NEVER)], 'en')
  expect(both).toContain("can't be reached; showing what Mortar saved (")
  expect(both).toContain('GitHub as of ')
  expect(both).toContain('nothing saved yet from Nexus')
  expect(offlineMessage([down('nexus', NEVER)], 'en')).toBe(
    "Nexus can't be reached; nothing saved to show yet",
  )
})
