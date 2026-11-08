import { afterEach, expect, mock, spyOn, test } from 'bun:test'
import { Clipboard, Events } from '@wailsio/runtime'
import * as launchsvc from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type { Result } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import * as profileService from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'

const { I18nProvider } = await import('@lingui/react')
const { cleanup, fireEvent, render, screen, waitFor } = await import('@testing-library/react')
const { usePageActionsSlot } = await import('../game/pageActions.ts')
const { i18n } = await import('../i18n/index.ts')
const { useProfiles } = await import('../profiles/store.ts')
const { testProfile } = await import('../profiles/testProfile.ts')
const { stubBinding } = await import('../testing/stubBinding.ts')
const { ProblemsTab } = await import('./ProblemsTab.tsx')
const { useMods } = await import('./store.ts')

const emptyResult = (): Result => ({
  duplicates: [],
  broken: [],
  missing: [],
  assetConflicts: [],
  settings: [],
  runErrors: [],
  drift: [],
  dismissed: [],
  unknown: false,
})

afterEach(() => {
  cleanup()
  mock.restore()
  useMods.setState(useMods.getInitialState(), true)
  useProfiles.setState(useProfiles.getInitialState(), true)
  usePageActionsSlot.setState({ slot: null })
})

function mountProblems(result: Result) {
  const slot = document.createElement('div')
  document.body.append(slot)
  usePageActionsSlot.setState({ slot })
  spyOn(Events, 'On').mockReturnValue(() => undefined)
  const copied: string[] = []
  spyOn(Clipboard, 'SetText').mockImplementation((text: string) => {
    copied.push(text)
    return Promise.resolve() as never
  })
  useProfiles.setState({
    game: { id: 'lethalcompany' } as never,
    openId: 'p1',
    profiles: [testProfile({ id: 'p1', name: 'Main' })],
  })
  useMods.setState({
    problems: result,
    problemsFor: 'p1',
    problemsSeed: false,
    modsFor: 'p1',
    mods: [],
    loaded: true,
  })
  render(
    <I18nProvider i18n={i18n}>
      <ProblemsTab />
    </I18nProvider>,
  )
  return copied
}

async function copyReport(copied: string[]) {
  const button = await waitFor(() => {
    const found = screen.getByRole('button', { name: 'Copy report' }) as HTMLButtonElement
    expect(found.disabled).toBe(false)
    return found
  })
  fireEvent.click(button)
  await waitFor(() => {
    expect(copied.length).toBe(1)
  })
  return copied[0] ?? ''
}

test('copy report includes a file conflict when that is the only finding', async () => {
  stubBinding(profileService, 'FileConflicts', () => [
    { path: 'BepInEx/plugins/x.dll', winner: 'keep', beaten: ['lose'] },
  ])
  stubBinding(launchsvc, 'StartupReports', () => [])
  const text = await copyReport(mountProblems(emptyResult()))
  expect(text).toContain('BepInEx/plugins/x.dll')
})

test('copy report includes a slow startup when that is the only finding', async () => {
  stubBinding(profileService, 'FileConflicts', () => [])
  stubBinding(launchsvc, 'StartupReports', () => [
    {
      mods: [
        {
          id: 'Slow.Mod',
          name: 'SlowMod',
          entryMs: 0,
          assetMs: 0,
          loadMs: 0,
          sampleMs: 0,
          patchMs: 0,
          eventMs: { GameLaunched: 3100 },
          packs: [],
        },
      ],
    },
  ])
  const text = await copyReport(mountProblems(emptyResult()))
  expect(text).toContain('SlowMod')
})

test('copy report includes a replaced file when that is the only finding', async () => {
  stubBinding(profileService, 'FileConflicts', () => [])
  stubBinding(launchsvc, 'StartupReports', () => [])
  const text = await copyReport(
    mountProblems({ ...emptyResult(), replaced: [{ path: 'Content/x.xnb', size: 12 }] }),
  )
  expect(text).toContain('Content/x.xnb')
})
