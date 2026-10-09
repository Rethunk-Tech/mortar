import { expect, test } from 'bun:test'
import { answerTargets, graphicsOptions } from './graphicsAsk.ts'

const graphics = {
  explanation: '',
  recommended: 'dx12',
  reason: '',
  source: '',
  choices: [
    { id: 'vulkan', label: 'Vulkan', args: null },
    { id: 'dx12', label: 'DirectX 12', args: ['-dx12'] },
  ],
}

test('the recommended choice leads and is the only one marked', () => {
  expect(graphicsOptions(graphics)).toEqual([
    { id: 'dx12', label: 'DirectX 12', recommended: true },
    { id: 'vulkan', label: 'Vulkan', recommended: false },
  ])
  expect(graphicsOptions({ ...graphics, choices: null })).toEqual([])
})

test('an answer also rewrites a profile override that would outrank it', () => {
  expect(answerTargets(undefined).profileOverride).toBe(false)
  expect(answerTargets({ skipIntro: 'true' }).profileOverride).toBe(false)
  expect(answerTargets({ graphicsApi: 'vulkan' }).profileOverride).toBe(true)
})
