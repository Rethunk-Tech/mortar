import { expect, test } from 'bun:test'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/dotnet/models.ts'
import { hookSnippet } from './hook.ts'

test('a hook passes self for an instance method and returns what the original returns', () => {
  const update = {
    type: 'GameNetcodeStuff.PlayerControllerB',
    kind: 'method',
    method: 'Update',
    static: false,
    void: true,
    params: [],
  }
  expect(hookSnippet(update as Member)).toBe(
    'On.GameNetcodeStuff.PlayerControllerB.Update += (orig, self) =>\n{\n    orig(self);\n};\n',
  )
  const nested = {
    type: 'A.Outer+Inner',
    kind: 'method',
    method: 'Pick',
    static: true,
    void: false,
    params: ['x'],
  }
  expect(hookSnippet(nested as Member)).toBe(
    'On.A.Outer.Inner.Pick += (orig, x) =>\n{\n    return orig(x);\n};\n',
  )
})
