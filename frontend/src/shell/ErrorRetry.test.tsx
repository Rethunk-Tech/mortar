import { expect, test } from 'bun:test'
import { I18nProvider } from '@lingui/react'
import { renderToStaticMarkup } from 'react-dom/server'
import { i18n } from '../i18n/index.ts'
import { ErrorRetry } from './ErrorRetry.tsx'

test('a signed-out error offers sign-in instead of retry', () => {
  const html = renderToStaticMarkup(
    <I18nProvider i18n={i18n}>
      <ErrorRetry
        error={{ message: 'x', details: 'cause', signedOut: true }}
        onRetry={() => undefined}
      />
    </I18nProvider>,
  )
  expect(html).toContain('Sign in')
  expect(html).not.toContain('Retry')
})

test('any other inline error offers retry with the cause on the title', () => {
  const html = renderToStaticMarkup(
    <I18nProvider i18n={i18n}>
      <ErrorRetry
        error={{ message: 'Could not read', details: 'disk' }}
        onRetry={() => undefined}
      />
    </I18nProvider>,
  )
  expect(html).toContain('Could not read')
  expect(html).toContain('Retry')
  expect(html).toContain('disk')
})
