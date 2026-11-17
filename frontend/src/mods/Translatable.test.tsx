import { afterEach, expect, mock, test } from 'bun:test'

const { I18nProvider } = await import('@lingui/react')
const { cleanup, fireEvent, render, screen, waitFor } = await import('@testing-library/react')
const { i18n } = await import('../i18n/index.ts')
const browse = await import(
  '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
)
const { stubBinding } = await import('../testing/stubBinding.ts')
const { Translatable } = await import('./Translatable.tsx')

afterEach(() => {
  cleanup()
  mock.restore()
})

const view = (source: string) => (
  <I18nProvider i18n={i18n}>
    <Translatable source={source}>{(text) => <p>{text}</p>}</Translatable>
  </I18nProvider>
)

test('Translate is optional per description and Show original restores the source-site copy', async () => {
  stubBinding(browse, 'TranslateDescription', (text: string, locale: string) => {
    if (text === 'Hola mundo' && locale === 'en') {
      return 'Hello world'
    }
    return `other:${text}`
  })

  render(
    <>
      {view('Hola mundo')}
      {view('Bonjour')}
    </>,
  )
  expect(screen.getByText('Hola mundo')).not.toBeNull()
  expect(screen.getByText('Bonjour')).not.toBeNull()
  const buttons = screen.getAllByRole('button', { name: 'Translate' })
  fireEvent.click(buttons[0] as HTMLElement)
  await waitFor(() => {
    expect(screen.getByText('Hello world')).not.toBeNull()
  })
  expect(screen.getByText('Bonjour')).not.toBeNull()
  expect(screen.queryByText('Hola mundo')).toBeNull()
  fireEvent.click(screen.getByRole('button', { name: 'Show original' }))
  expect(screen.getByText('Hola mundo')).not.toBeNull()
  expect(screen.getByText('Bonjour')).not.toBeNull()
})

test('a signed-out saved listing still shows its source-site copy before anyone translates', () => {
  render(view('Saved listing copy'))
  expect(screen.getByText('Saved listing copy')).not.toBeNull()
  expect(screen.getByRole('button', { name: 'Translate' })).not.toBeNull()
})
