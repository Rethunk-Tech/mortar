// wails3 emits `export enum` for Go named-string types. erasableSyntaxOnly forbids enums,
// so each generated enum becomes a const object plus a value-union type.
import { join } from 'node:path'

const ROOT = join(import.meta.dirname, '../frontend/bindings')

export function rewriteEnums(src: string): string {
  return src.replace(/export enum (\w+) \{([\s\S]*?)\n\};?/g, (_, name: string, body: string) => {
    const members = body.replace(/(\w+) = /g, '$1: ')
    return `export const ${name} = {${members}} as const\nexport type ${name} = (typeof ${name})[keyof typeof ${name}]`
  })
}

if (import.meta.main) {
  for (const file of new Bun.Glob('**/*.ts').scanSync({ cwd: ROOT, absolute: true })) {
    const before = await Bun.file(file).text()
    const after = rewriteEnums(before)
    if (after !== before) {
      await Bun.write(file, after)
    }
  }
}
