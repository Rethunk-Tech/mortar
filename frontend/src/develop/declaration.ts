const CONSTRUCTOR_NAME = /^\.c?ctor$/
const TYPE_PARTS = /[.+]/
const ESCAPE = /[.*+?^${}()|[\]\\]/g

// The line declaring a member in decompiled C# or IL, or -1. A method is its name before a parameter list (IL writes a
// space before it) on a line that is not a call. A field is its name after a type, ending the line or before ; or =,
// which a use of the field inside a method body (after a dot, an operator or `return`) does not match first, since
// fields are declared above the methods.
export function declarationLine(source: string, type: string, kind: string, name: string): number {
  const lines = source.split('\n')
  if (kind === 'field') {
    const declared = new RegExp(`[\\w>\\]]\\s+${name.replace(ESCAPE, '\\$&')}\\s*(;|=(?!=)|$)`)
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
