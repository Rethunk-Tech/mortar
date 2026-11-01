import { describe, expect, test } from 'bun:test'
import { declarationLine } from './declaration.ts'

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
