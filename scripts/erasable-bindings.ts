// wails3 emits `export enum` for Go named-string types. erasableSyntaxOnly forbids enums,
// so each generated enum becomes a const object plus a value-union type.
import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const ROOT = join(import.meta.dirname, '../frontend/bindings')

export function rewriteEnums(src: string): string {
  return src.replace(/export enum (\w+) \{([\s\S]*?)\n\};?/g, (_, name: string, body: string) => {
    const members = body.replace(/(\w+) = /g, '$1: ')
    return `export const ${name} = {${members}} as const\nexport type ${name} = (typeof ${name})[keyof typeof ${name}]`
  })
}

function walk(dir: string): string[] {
  const out: string[] = []
  for (const ent of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, ent.name)
    if (ent.isDirectory()) {
      out.push(...walk(p))
    } else if (ent.name.endsWith('.ts')) {
      out.push(p)
    }
  }
  return out
}

if (import.meta.main) {
  for (const file of walk(ROOT)) {
    const before = readFileSync(file, 'utf8')
    const after = rewriteEnums(before)
    if (after !== before) {
      writeFileSync(file, after)
    }
  }
}
