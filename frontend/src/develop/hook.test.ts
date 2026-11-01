import { expect, test } from 'bun:test'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/dotnet/models.ts'
import { harmonyPatchSnippet, hookSnippet, monoModHookSnippet } from './hook.ts'

test('a hook passes self for an instance method and returns what the original returns', () => {
  const update = {
    type: 'GameNetcodeStuff.PlayerControllerB',
    kind: 'method',
    method: 'Update',
    static: false,
    void: true,
    params: [],
    types: [],
    returns: 'void',
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
    types: ['int'],
    returns: 'int',
  }
  expect(hookSnippet(nested as Member)).toBe(
    'On.A.Outer.Inner.Pick += (orig, x) =>\n{\n    return orig(x);\n};\n',
  )
})

const damage = {
  type: 'GameNetcodeStuff.PlayerControllerB',
  kind: 'method',
  method: 'DamagePlayer',
  static: false,
  void: true,
  params: ['damageNumber', 'force'],
  types: ['int', 'ref UnityEngine.Vector3'],
  returns: 'void',
} as Member

test('a manual MonoMod hook names the overload and passes a by-ref argument with ref', () => {
  expect(monoModHookSnippet(damage)).toBe(
    [
      'private static Hook? damagePlayerHook;',
      '',
      '// In Awake:',
      'damagePlayerHook = new Hook(typeof(GameNetcodeStuff.PlayerControllerB).GetMethod("DamagePlayer", BindingFlags.Instance | BindingFlags.Public | BindingFlags.NonPublic, null, new[] { typeof(int), typeof(UnityEngine.Vector3).MakeByRefType() }, null), DamagePlayer);',
      '',
      'private delegate void orig_DamagePlayer(GameNetcodeStuff.PlayerControllerB self, int damageNumber, ref UnityEngine.Vector3 force);',
      '',
      'private static void DamagePlayer(orig_DamagePlayer orig, GameNetcodeStuff.PlayerControllerB self, int damageNumber, ref UnityEngine.Vector3 force)',
      '{',
      '    orig(self, damageNumber, ref force);',
      '}',
      '',
    ].join('\n'),
  )
})

test('a Harmony patch targets the overload and takes the instance and arguments by name', () => {
  expect(harmonyPatchSnippet(damage)).toBe(
    [
      '[HarmonyPatch(typeof(GameNetcodeStuff.PlayerControllerB), "DamagePlayer", new[] { typeof(int), typeof(UnityEngine.Vector3).MakeByRefType() })]',
      'internal static class PlayerControllerBDamagePlayerPatch',
      '{',
      '    private static void Prefix(GameNetcodeStuff.PlayerControllerB __instance, int damageNumber, ref UnityEngine.Vector3 force)',
      '    {',
      '    }',
      '}',
      '',
    ].join('\n'),
  )
})

const cctor = {
  ...damage,
  method: '.cctor',
  static: true,
  params: [],
  types: [],
  returns: 'void',
} as Member
const ctor = { ...cctor, method: '.ctor', static: false } as Member

test('a static constructor is told apart from an instance one in both snippets', () => {
  expect(harmonyPatchSnippet(cctor)).toContain('MethodType.StaticConstructor, Type.EmptyTypes')
  expect(harmonyPatchSnippet(ctor)).toContain('MethodType.Constructor, Type.EmptyTypes')
  expect(monoModHookSnippet(cctor)).toContain('private static Hook? cctorHook;')
  expect(monoModHookSnippet(ctor)).toContain('private static Hook? ctorHook;')
})
