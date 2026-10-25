import type { Member } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/dotnet/models.ts'

// HookGen names a constructor's hook ctor (and a static constructor's cctor), without the dot.
const CONSTRUCTOR = /^\.c?ctor$/

/** A MonoMod HookGen hook (an On. event from HookGenPatcher's MMHOOK assembly) that calls the original method. */
export function hookSnippet(m: Member): string {
  const method = m.method.replace(CONSTRUCTOR, (c) => c.slice(1))
  const args = m.static ? (m.params ?? []) : ['self', ...(m.params ?? [])]
  const call = `orig(${args.join(', ')});`
  return [
    `On.${m.type.replaceAll('+', '.')}.${method} += (${['orig', ...args].join(', ')}) =>`,
    '{',
    `    ${m.void ? call : `return ${call}`}`,
    '};',
    '',
  ].join('\n')
}
