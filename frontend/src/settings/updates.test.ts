import { expect, test } from 'bun:test'
import type { Info } from '../../bindings/github.com/Rethunk-AI/mortar/internal/updatesvc/models.ts'
import { useMortarUpdate } from './updates.ts'

test('checking again keeps a found or staged update on offer', async () => {
  for (const phase of ['available', 'ready'] as const) {
    useMortarUpdate.setState(
      {
        ...useMortarUpdate.getInitialState(),
        info: { version: '1.0.0', off: '' } satisfies Info,
        phase,
      },
      true,
    )
    await useMortarUpdate.getState().check()
    expect(useMortarUpdate.getState().phase).toBe(phase)
  }
})
