import { expect, test } from 'bun:test'
import { seedPreview } from './seedPreview.ts'

test('a nearby Mortar’s payload is previewed, under the file tab', async () => {
  const calls: string[] = []
  const record = (name: string) => async (value?: unknown) => {
    calls.push(`${name}:${String(value ?? '')}`)
  }
  await seedPreview(
    { profileId: '', tab: 'data', seed: 'cGF5bG9hZA', run: 1 },
    {
      setTab: (tab) => calls.push(`tab:${tab}`),
      setText: (text) => calls.push(`text:${text}`),
      previewLink: record('link'),
      previewFile: record('file'),
      previewData: record('data'),
      previewExternal: record('external'),
      previewCollectionUpdate: record('collection'),
    },
  )
  expect(calls).toEqual(['tab:file', 'data:cGF5bG9hZA'])
})
