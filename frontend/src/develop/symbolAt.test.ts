import { expect, test } from 'bun:test'
import { csArity, identifierAt, ilArity, ilOwner } from './symbolAt.ts'

test('identifierAt finds the name a click lands in, and nothing on punctuation', () => {
  const line = 'public Camera gameplayCamera;'
  expect(identifierAt(line, 16)).toBe('gameplayCamera')
  // Just past the last letter, on the right half of the 'a'.
  expect(identifierAt(line, 28)).toBe('gameplayCamera')
  expect(identifierAt(line, 8)).toBe('Camera')
  expect(identifierAt('a + b', 2)).toBe('')
})

test('ilOwner reads the type a call or field access names', () => {
  expect(
    ilOwner(
      'IL_0012: call instance void DungeonGenerator::RefreshVisibility()',
      'RefreshVisibility',
    ),
  ).toBe('DungeonGenerator')
  expect(
    ilOwner(
      'ldfld class [UnityEngine]UnityEngine.Camera GameNetcodeStuff.PlayerControllerB::gameplayCamera',
      'gameplayCamera',
    ),
  ).toBe('GameNetcodeStuff.PlayerControllerB')
  expect(ilOwner('call instance void Outer/Inner::Go()', 'Go')).toBe('Outer+Inner')
  expect(
    ilOwner(
      'callvirt instance void class [mscorlib]System.Collections.Generic.List`1<int32>::Add(!0)',
      'Add',
    ),
  ).toBe('System.Collections.Generic.List`1')
  expect(ilOwner('call instance void Thing::Updated()', 'Update')).toBe('')
})

test("ilArity counts a call's arguments, not those of its generic types", () => {
  expect(
    ilArity(
      'call instance void AdjacentRoomCullingModified::UpdateRendererLists()',
      'UpdateRendererLists',
    ),
  ).toBe(0)
  expect(
    ilArity(
      'call instance void AdjacentRoomCullingModified::UpdateRendererLists(class [mscorlib]System.Collections.Generic.List`1<class [DunGen]DunGen.Tile>, class [mscorlib]System.Collections.Generic.List`1<class [DunGen]DunGen.Door>)',
      'UpdateRendererLists',
    ),
  ).toBe(2)
  expect(ilArity('ldfld class Tile AdjacentRoomCullingModified::currentTile', 'currentTile')).toBe(
    -1,
  )
})

test("csArity counts a declaration's parameters", () => {
  expect(
    csArity(
      '\tpublic void UpdateRendererLists(List<Tile> tiles, List<Door> doors)',
      'UpdateRendererLists',
    ),
  ).toBe(2)
  expect(csArity('\tpublic void UpdateRendererLists()', 'UpdateRendererLists')).toBe(0)
  expect(csArity('\tpublic int count;', 'count')).toBe(-1)
})
