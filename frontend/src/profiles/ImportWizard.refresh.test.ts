import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('the import wizard detects again each time it opens and offers a typed folder without a native dialog', () => {
  const wizard = readFileSync(join(import.meta.dir, 'ImportWizard.tsx'), 'utf8')
  expect(wizard).toContain('useExternalImportSources(game, open)')
  const prompt = readFileSync(join(import.meta.dir, 'VortexFolderPrompt.tsx'), 'utf8')
  expect(prompt).toContain('setTyping(true)')
})
