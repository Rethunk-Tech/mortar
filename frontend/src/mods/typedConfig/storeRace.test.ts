import { expect, mock, test } from 'bun:test'
import type { Target } from './store.ts'

let releaseFirst: () => void = () => undefined
mock.module('./api.ts', () => ({
  configApi: {
    files: async (t: Target) => {
      if (t.key === 'slow') {
        await new Promise<void>((resolve) => {
          releaseFirst = resolve
        })
        return [{ name: 'Slow.cfg', label: '', format: 'cfg', sections: [] }]
      }
      return [{ name: 'BepInEx.cfg', label: '', format: 'cfg', sections: [] }]
    },
    schema: async () => [],
  },
}))

const { useTypedConfig } = await import('./store.ts')

test('a mod opened while another still loads keeps the pane when the first answer arrives late', async () => {
  const { open } = useTypedConfig.getState()
  const first = open({ game: 'lethal-company', profile: 'p', key: 'slow', id: '' })
  await open({ game: 'lethal-company', profile: 'p', key: '', id: '' }, 'BepInEx.cfg')
  releaseFirst()
  await first
  const s = useTypedConfig.getState()
  expect(s.current).toBe('BepInEx.cfg')
  expect(s.files.map((f) => f.name)).toEqual(['BepInEx.cfg'])
})
