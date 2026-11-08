import type { Member } from '../../bindings/github.com/Rethunk-Tech/go-dotnet/models.ts'

// HookGen names a constructor's hook ctor (and a static constructor's cctor), without the dot.
const CONSTRUCTOR = /^\.c?ctor$/
const REF = 'ref '

const ownerName = (m: Member) => m.type.replaceAll('+', '.')
const isConstructor = (m: Member) => CONSTRUCTOR.test(m.method)
// A name a handler or patch class can carry: a constructor's is Ctor, a static constructor's Cctor.
const handlerName = (m: Member) => {
  if (!isConstructor(m)) {
    return m.method
  }
  return m.method === '.cctor' ? 'Cctor' : 'Ctor'
}
const typesOf = (m: Member) => m.types ?? []
const paramsOf = (m: Member) => m.params ?? []

/** A MonoMod HookGen hook (an On. event from HookGenPatcher's MMHOOK assembly) that calls the original method. */
export function hookSnippet(m: Member): string {
  const method = m.method.replace(CONSTRUCTOR, (c) => c.slice(1))
  const args = m.static ? paramsOf(m) : ['self', ...paramsOf(m)]
  const call = `orig(${args.join(', ')});`
  return [
    `On.${ownerName(m)}.${method} += (${['orig', ...args].join(', ')}) =>`,
    '{',
    `    ${m.void ? call : `return ${call}`}`,
    '};',
    '',
  ].join('\n')
}

// typeof for a parameter type, by-ref ones as Reflection names them.
function typeOf(t: string): string {
  return t.startsWith(REF) ? `typeof(${t.slice(REF.length)}).MakeByRefType()` : `typeof(${t})`
}

function typeArray(m: Member): string {
  const types = typesOf(m)
  return types.length === 0 ? 'Type.EmptyTypes' : `new[] { ${types.map(typeOf).join(', ')} }`
}

// The parameters as declared and as passed on: a by-ref one passes with ref.
function declared(m: Member): string[] {
  return paramsOf(m).map((p, i) => `${typesOf(m)[i] ?? 'object'} ${p}`)
}

function passed(m: Member): string[] {
  return paramsOf(m).map((p, i) => ((typesOf(m)[i] ?? '').startsWith(REF) ? `ref ${p}` : p))
}

/**
 * A MonoMod RuntimeDetour Hook built by hand, kept in a static field as the template keeps its own: a delegate for the
 * original, the handler that calls it, and the line that installs it in Awake. It names the parameter types, so it finds
 * the right overload and needs no MMHOOK assembly.
 */
export function monoModHookSnippet(m: Member): string {
  const owner = ownerName(m)
  const name = handlerName(m)
  const self = m.static || isConstructor(m) ? [] : [`${owner} self`]
  const args = m.static || isConstructor(m) ? passed(m) : ['self', ...passed(m)]
  const returns = isConstructor(m) || m.void ? 'void' : m.returns
  const flags = `BindingFlags.${m.static ? 'Static' : 'Instance'} | BindingFlags.Public | BindingFlags.NonPublic`
  const target = isConstructor(m)
    ? `typeof(${owner}).GetConstructor(${flags}, null, ${typeArray(m)}, null)`
    : `typeof(${owner}).GetMethod("${m.method}", ${flags}, null, ${typeArray(m)}, null)`
  const call = `orig(${args.join(', ')});`
  const field = `${name.charAt(0).toLowerCase()}${name.slice(1)}Hook`
  return [
    `private static Hook? ${field};`,
    '',
    '// In Awake:',
    `${field} = new Hook(${target}, ${name});`,
    '',
    `private delegate ${returns} orig_${name}(${[...self, ...declared(m)].join(', ')});`,
    '',
    `private static ${returns} ${name}(${[`orig_${name} orig`, ...self, ...declared(m)].join(', ')})`,
    '{',
    `    ${returns === 'void' ? call : `return ${call}`}`,
    '}',
    '',
  ].join('\n')
}

/** A Harmony patch class whose Prefix runs before the method, given the instance and its arguments by name. */
export function harmonyPatchSnippet(m: Member): string {
  const owner = ownerName(m)
  const target = isConstructor(m)
    ? `typeof(${owner}), MethodType.${m.method === '.cctor' ? 'StaticConstructor' : 'Constructor'}, ${typeArray(m)}`
    : `typeof(${owner}), "${m.method}", ${typeArray(m)}`
  const instance = m.static ? [] : [`${owner} __instance`]
  const args = paramsOf(m).map((p, i) => {
    const t = typesOf(m)[i] ?? 'object'
    return t.startsWith(REF) ? `ref ${t.slice(REF.length)} ${p}` : `${t} ${p}`
  })
  return [
    `[HarmonyPatch(${target})]`,
    `internal static class ${owner.split('.').at(-1) ?? owner}${handlerName(m)}Patch`,
    '{',
    `    private static void Prefix(${[...instance, ...args].join(', ')})`,
    '    {',
    '    }',
    '}',
    '',
  ].join('\n')
}

// A member's identity among overloads: its type, name and parameter names.
export const memberKey = (m: Member) => `${m.type}::${m.method}(${(m.params ?? []).join(',')})`
