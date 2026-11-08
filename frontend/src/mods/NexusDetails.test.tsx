import { afterEach, expect, test } from 'bun:test'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

const { I18nProvider } = await import('@lingui/react')
const { cleanup, render } = await import('@testing-library/react')
const { i18n } = await import('../i18n/index.ts')
const { useNexusDetails } = await import('./nexusDetails.ts')
const { NexusDetails } = await import('./NexusDetails.tsx')

const details = {
  page: {
    version: '2.0',
    uploadedBy: 'someone',
    uploaderUrl: '',
    endorsements: 12,
    downloads: 1234,
    uniqueDownloads: 567,
    created: '2025-01-02T00:00:00Z',
    updated: '2025-03-04T00:00:00Z',
    summary: '',
    description: '',
    adult: false,
    status: 'published',
    available: true,
    endorsement: {},
    requirements: [],
  },
  category: 'NexusRawCategory',
  files: [],
  changelogs: [],
} as unknown as Details

const mod = { version: '1.0' } as unknown as Mod

const html = (inPanel: boolean) => {
  useNexusDetails.setState({ byKey: { ':7': { details } } })
  const { container } = render(
    <I18nProvider i18n={i18n}>
      <NexusDetails mod={mod} modId={7} fileId={0} inPanel={inPanel} />
    </I18nProvider>,
  )
  return container.innerHTML
}

afterEach(() => {
  cleanup()
  useNexusDetails.setState(useNexusDetails.getInitialState())
})

test('in the details panel the Nexus page shows no second copy of the facts the sidebar already gives', () => {
  const out = html(true)
  expect(out).toContain('Uploaded by')
  expect(out).toContain('unique')
  expect(out).not.toContain('Latest')
  expect(out).not.toContain('NexusRawCategory')
  expect(out).not.toContain('Endorsements')
})

test('the wide panel keeps the whole facts block', () => {
  const out = html(false)
  expect(out).toContain('Latest')
  expect(out).toContain('NexusRawCategory')
  expect(out).toContain('Endorsements')
  expect(out).toContain('Uploaded by')
})
