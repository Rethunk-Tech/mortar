import { expect, test } from 'bun:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { MarkdownView } from './MarkdownView.tsx'

const html = (source: string) => renderToStaticMarkup(<MarkdownView source={source} />)

test('markdown renders, and a badge inside a link renders as a linked picture, not raw brackets', () => {
  const out = html(
    '[![icon](https://cdn.modrinth.com/data/x/icon.png)](https://thunderstore.io/c/x/p/a/b)',
  )
  expect(out).toContain('/mod-picture/?u=https%3A%2F%2Fcdn.modrinth.com%2Fdata%2Fx%2Ficon.png')
  expect(out).not.toContain('](')
  const doc = html('# Title\n\n- one\n- two\n\n| a | b |\n|---|---|\n| 1 | 2 |')
  expect(doc).toContain('<h1>Title</h1>')
  expect(doc).toContain('<li>one</li>')
  expect(doc).toContain('<td>1</td>')
})

test('raw HTML and non-web links never reach the page as markup', () => {
  const out = html(
    '<script>alert(1)</script>\n\n[x](javascript:alert(1)) ![y](http://plain.example/a.png)',
  )
  expect(out).not.toContain('<script')
  expect(out).not.toContain('&lt;')
  expect(out).not.toContain('javascript:')
  expect(out).not.toContain('plain.example')
  expect(out).not.toContain('<img')
})
