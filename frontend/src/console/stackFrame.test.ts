import { expect, test } from 'bun:test'
import { parseFrame } from './stackFrame.ts'

test('parseFrame reads each runtime format and where its name sits', () => {
  // A Unity exception from a Lethal Company run.
  const unity = 'CentipedeAI.OnDestroy () (at <aca1e98d6f844d3f85cd22b6f7be2f9b>:0)'
  expect(parseFrame(unity)).toEqual({
    type: 'CentipedeAI',
    method: 'OnDestroy',
    params: 0,
    mvid: 'aca1e98d6f844d3f85cd22b6f7be2f9b',
    start: 0,
    end: 21,
  })
  const colon = parseFrame('MortarBepInExBridge.LogSources:LogException(Exception, Object)')
  expect([colon?.type, colon?.method, colon?.params]).toEqual([
    'MortarBepInExBridge.LogSources',
    'LogException',
    2,
  ])
  const mono =
    '  at LateCompany.Patches.ConnectionApproval_Patch.Postfix (Unity.Netcode.NetworkManager request) [0x00019] in <9c56>:0 '
  const f = parseFrame(mono)
  expect([f?.type, f?.method, f?.params, mono.slice(f?.start, f?.end)]).toEqual([
    'LateCompany.Patches.ConnectionApproval_Patch',
    'Postfix',
    1,
    'LateCompany.Patches.ConnectionApproval_Patch.Postfix',
  ])
  // Harmony's wrapper names the method it stands in for.
  const dmd = parseFrame(
    '  at (wrapper dynamic-method) GameNetworkManager.DMD<GameNetworkManager::ConnectionApproval>(GameNetworkManager)',
  )
  expect([dmd?.type, dmd?.method, dmd?.params]).toEqual([
    'GameNetworkManager',
    'ConnectionApproval',
    1,
  ])
  expect(
    parseFrame('NullReferenceException: Object reference not set to an instance of an object'),
  ).toBeNull()
  expect(parseFrame('Stack trace:')).toBeNull()
})
