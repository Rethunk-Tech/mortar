// wails3 emits `export enum` for Go named-string types. erasableSyntaxOnly forbids enums,
// so each generated enum becomes a const object plus a value-union type.
import { globSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const ROOT = join(import.meta.dirname, '../frontend/bindings')

export function rewriteEnums(src: string): string {
  return src.replace(/export enum (\w+) \{([\s\S]*?)\n\};?/g, (_, name: string, body: string) => {
    const members = body.replace(/(\w+) = /g, '$1: ')
    return `export const ${name} = {${members}} as const\nexport type ${name} = (typeof ${name})[keyof typeof ${name}]`
  })
}

if (import.meta.main) {
  // Node APIs only: the Flathub build runs this with the SDK's node, which has no Bun global.
  for (const file of globSync('**/*.ts', { cwd: ROOT })) {
    const path = join(ROOT, file)
    const before = readFileSync(path, 'utf8')
    const after = rewriteEnums(before)
    if (after !== before) {
      writeFileSync(path, after)
    }
  }
}
