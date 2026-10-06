import { expect, mock, test } from 'bun:test'

mock.module('@lingui/core/macro', () => ({
  msg: (parts: TemplateStringsArray, ...values: unknown[]) => String.raw({ raw: parts }, ...values),
  plural: (n: number, forms: { other: string }) => forms.other.replace('#', String(n)),
}))

const { offlineMessage, unreachableNote, updatesReasonOf } = await import('./offlineText.ts')

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
  expect(offlineMessage([down('nexus', NEVER), down('github', NEVER)], 'en')).toBe(
    "GitHub and Nexus can't be reached; nothing saved to show yet",
  )
  expect(unreachableNote(['nexus', 'github'])).toBe(unreachableNote(['github', 'nexus']))
  expect(unreachableNote(['nexus', 'github'])).toBe("GitHub and Nexus can't be reached")
  expect(offlineMessage([down('nexus', NEVER)], 'en')).toBe(
    "Nexus can't be reached; nothing saved to show yet",
  )
})

test('a skipped update check says why without promising saved data', () => {
  const reason = updatesReasonOf([down('nexus', NEVER), down('github', '2026-10-05T12:00:00Z')])
  expect(reason).toBe("GitHub and Nexus can't be reached, so Mortar can't check for mod updates")
  expect(
    updatesReasonOf([down('nexus', NEVER), { ...down('github', NEVER), unreachable: false }]),
  ).toBe('')
})
