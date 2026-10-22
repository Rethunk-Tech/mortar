import { expect, test } from 'bun:test'
import { formatter } from '@lingui/format-po'
import { withoutFuzzy } from '../../lingui.config.ts'

const PO = `msgid ""
msgstr ""
"Language: es\\n"

#, fuzzy
msgid "No saves match"
msgstr "Ninguna partida coincide"

msgid "Close"
msgstr "Cerrar"
`

test('compiling leaves out a translation Weblate marks as needing editing', async () => {
  const format = withoutFuzzy(formatter({ lineNumbers: false }))
  const catalog = await format.parse(PO, { locale: 'es', sourceLocale: 'en', filename: 'es.po' })
  const byMessage = Object.fromEntries(
    Object.values(catalog).map((m) => [m.message, m.translation]),
  )
  expect(byMessage).toEqual({ 'No saves match': '', Close: 'Cerrar' })
})
