import { expect, test } from 'bun:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { DisabledReason } from './DisabledReason.tsx'

const remove = 'Remove'
const go = 'Go'

test('disabled children stay in a span so the tooltip can attach', () => {
  const html = renderToStaticMarkup(
    <DisabledReason title="Stop the game to change mods." disabled={true}>
      <button type="button" disabled={true}>
        {remove}
      </button>
    </DisabledReason>,
  )
  expect(html).toContain('span')
  expect(html).toContain('Remove')
})

test('enabled children are not wrapped', () => {
  const html = renderToStaticMarkup(
    <DisabledReason title="Stop the game to change mods." disabled={false}>
      <button type="button">{go}</button>
    </DisabledReason>,
  )
  expect(html).toBe('<button type="button">Go</button>')
})
