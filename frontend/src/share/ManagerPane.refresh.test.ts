import { expect, test } from 'bun:test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

test('the manager list detects again each time it opens and offers a typed folder without a native dialog', () => {
  const pane = readFileSync(join(import.meta.dir, 'ManagerPane.tsx'), 'utf8')
  expect(pane).toContain('useExternalImportSources(game, true)')
  const prompt = readFileSync(join(import.meta.dir, '../profiles/VortexFolderPrompt.tsx'), 'utf8')
  expect(prompt).toContain('setTyping(true)')
})
