import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('launch preview renders a load error instead of swallowing PreviewCommand failures', () => {
  const src = readFileSync(join(import.meta.dir, 'LaunchPreview.tsx'), 'utf8')
  expect(src).not.toContain('.catch(() => undefined)')
  expect(src).toContain('setLoadError(errorMessage(e))')
  expect(src).toContain('{preview.error || loadError}')
})
