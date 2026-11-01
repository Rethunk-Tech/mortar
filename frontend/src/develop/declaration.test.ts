import { describe, expect, test } from 'bun:test'
import { declarationLine, declares } from './declaration.ts'

const csharp = [
  'public class PlayerControllerB : NetworkBehaviour',
  '{',
  '\tpublic Camera gameplayCamera;',
  '\tpublic float targetFOV = 66f;',
  '\tprivate void Update()',
  '\t{',
  '\t\tgameplayCamera.fieldOfView = targetFOV;',
  '\t\tUpdate();',
  '\t}',
  '\tpublic PlayerControllerB()',
].join('\n')

const il = [
  '.class public PlayerControllerB',
  '\t.field public class [UnityEngine]UnityEngine.Camera gameplayCamera',
  '\t.method private hidebysig instance void Update () cil managed',
].join('\n')

describe('declarationLine', () => {
  test('a field in C# and IL, not its use in a method', () => {
    expect(declarationLine(csharp, 'PlayerControllerB', 'field', 'gameplayCamera')).toBe(2)
    expect(declarationLine(csharp, 'PlayerControllerB', 'field', 'targetFOV')).toBe(3)
    expect(declarationLine(il, 'PlayerControllerB', 'field', 'gameplayCamera')).toBe(1)
  })

  test('a method and a constructor, not a call', () => {
    expect(declarationLine(csharp, 'PlayerControllerB', 'method', 'Update')).toBe(4)
    expect(declarationLine(il, 'PlayerControllerB', 'method', 'Update')).toBe(2)
    expect(declarationLine(csharp, 'GameNetcodeStuff.PlayerControllerB', 'method', '.ctor')).toBe(9)
  })
})

test('declares tells a declaration from a use', () => {
  expect(declares('\tprotected virtual void LateUpdate()', 'LateUpdate')).toBe(true)
  expect(declares('\tpublic Camera gameplayCamera;', 'gameplayCamera')).toBe(true)
  expect(declares('\t\tRefreshVisibility();', 'RefreshVisibility')).toBe(false)
  expect(declares('\t\tgameplayCamera.fieldOfView = 60f;', 'gameplayCamera')).toBe(false)
})
