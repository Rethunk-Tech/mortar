import { expect, test } from 'bun:test'

// Rows render one chip each, and a read started per row spends Nexus quota per mod. Hooks that enqueue a read
// (useNexusEntry, loadDetails) belong to the details panel.
test('the extra-files chip reads the details store and never starts a full read', async () => {
  const source = await Bun.file(new URL('./ExtraFilesChip.tsx', import.meta.url)).text()
  expect(source).not.toMatch(/useNexusEntry|loadDetails|primeDetails/)
  expect(source).toContain('useNexusDetails((s) => s.byId[nexusId]?.details?.files)')
})
