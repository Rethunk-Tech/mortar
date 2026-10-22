import { afterAll, expect, test } from 'bun:test'

const { I18nProvider } = await import('@lingui/react')
const { cleanup, render, screen } = await import('@testing-library/react')
const { i18n } = await import('../i18n/index.ts')
const { AddToGroupDialog } = await import('./AddToGroupDialog.tsx')

afterAll(() => {
  cleanup()
})

// MUI throws when a MenuItem renders outside a Menu or MenuList, which blanked the view for any profile with a group.
test('the dialog lists the groups a profile already has', () => {
  const profile = {
    id: 'p',
    name: 'Main',
    notes: '',
    cover: '',
    order: 0,
    created: '',
    updated: '',
    entries: [],
    groups: [{ name: 'Bosses', keys: [] }],
  }
  render(
    <I18nProvider i18n={i18n}>
      <AddToGroupDialog open={true} onClose={() => undefined} mods={[]} profile={profile} />
    </I18nProvider>,
  )
  expect(screen.getByRole('menuitem', { name: 'Bosses' })).toBeTruthy()
})
