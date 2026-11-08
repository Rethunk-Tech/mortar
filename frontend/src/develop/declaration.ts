import { escapeRegExp } from '../shell/escapeRegExp.ts'

const CONSTRUCTOR_NAME = /^\.c?ctor$/
const TYPE_PARTS = /[.+]/
// The line declaring a member in decompiled C# or IL, or -1. A method is its name before a parameter list (IL writes a
// space before it) on a line that is not a call. A field is its name after a type, ending the line or before ; or =,
// which a use of the field inside a method body (after a dot, an operator or `return`) does not match first, since
// fields are declared above the methods.
export function declarationLine(source: string, type: string, kind: string, name: string): number {
  const lines = source.split('\n')
  if (kind === 'field') {
    const declared = new RegExp(`[\\w>\\]]\\s+${escapeRegExp(name)}\\s*(;|=(?!=)|$)`)
    return lines.findIndex(
      (line) => declared.test(line.trimEnd()) && !line.trimStart().startsWith('return '),
    )
  }
  const method = CONSTRUCTOR_NAME.test(name) ? (type.split(TYPE_PARTS).at(-1) ?? '') : name
  return lines.findIndex(
    (line) =>
      (line.includes(` ${method}(`) || line.includes(` ${method} (`)) &&
      !line.trimEnd().endsWith(';'),
  )
}

// Whether a line of decompiled C# or IL declares a member called name, by the same rules declarationLine uses.
export function declares(line: string, name: string): boolean {
  return (
    declarationLine(line, '', 'method', name) === 0 ||
    declarationLine(line, '', 'field', name) === 0
  )
}

const CALLED = /([A-Za-z_]\w*)\s*(?:<[^>()]*>)?\s*\(/g
const NAMED = /([A-Za-z_]\w*)\s*(?:;|=(?!=)|$)/g
// Words a statement opens with before a parenthesis, which no member is named.
const STATEMENTS = new Set([
  'if',
  'for',
  'foreach',
  'while',
  'switch',
  'catch',
  'using',
  'lock',
  'return',
  'fixed',
  'nameof',
  'typeof',
  'sizeof',
  'new',
  'base',
  'this',
])

// ILSpy writes every member with its modifiers, which a local variable never has. In IL a field is a .field line and a
// method's name sits on the line its parameter list ends (cil managed) or opens on.
const MEMBER =
  /^\s*(?:public|private|protected|internal|static|extern|override|virtual|abstract|sealed|unsafe|readonly|const|event)\b/
const IL_FIELD = /^\s*\.field\b/
const IL_METHOD = /\(\s*$|\)\s*(?:cil|runtime)\s+managed\b/

/** The member a line of decompiled C# or IL declares, or null when it declares none (a local variable, a statement). */
export function declaredIn(line: string): { kind: 'method' | 'field'; name: string } | null {
  const member = MEMBER.test(line)
  for (const m of line.matchAll(CALLED)) {
    const name = m[1] ?? ''
    if (
      (member || IL_METHOD.test(line)) &&
      !STATEMENTS.has(name) &&
      declarationLine(line, '', 'method', name) === 0
    ) {
      return { kind: 'method', name }
    }
  }
  for (const m of line.trimEnd().matchAll(NAMED)) {
    const name = m[1] ?? ''
    if ((member || IL_FIELD.test(line)) && declarationLine(line, '', 'field', name) === 0) {
      return { kind: 'field', name }
    }
  }
  return null
}
