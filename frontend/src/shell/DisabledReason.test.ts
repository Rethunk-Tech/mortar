import { expect, test } from 'bun:test'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { DisabledReason } from './DisabledReason.tsx'

test('disabled children stay in a span so the tooltip can attach', () => {
  const html = renderToStaticMarkup(
    createElement(
      DisabledReason,
      { title: 'Stop the game to change mods.', disabled: true },
      createElement('button', { type: 'button', disabled: true }, 'Remove'),
    ),
  )
  expect(html).toContain('span')
  expect(html).toContain('Remove')
})

test('enabled children are not wrapped', () => {
  const html = renderToStaticMarkup(
    createElement(
      DisabledReason,
      { title: 'Stop the game to change mods.', disabled: false },
      createElement('button', { type: 'button' }, 'Go'),
    ),
  )
  expect(html).toBe('<button type="button">Go</button>')
})
