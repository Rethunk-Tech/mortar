// Every bound method's name and declared result, read from the generated bindings, so the demo can answer a call it
// has no recording for with an empty value of the right shape.
const sources = import.meta.glob<string>(
  [
    '../../bindings/github.com/Rethunk-Tech/mortar/internal/**/*.ts',
    '!**/models.ts',
    '!**/index.ts',
  ],
  { query: '?raw', import: 'default', eager: true },
)

const ROOT = '/internal/'
const BINDING =
  /export function (\w+)\([^)]*\): \$CancellablePromise<(.+)> \{\s+return \$Call\.ByID\((\d+)/g

interface Method {
  /** The service folder and method, as `profile.Rename`. */
  name: string
  result: string
}

const methods = new Map<number, Method>()
for (const [path, source] of Object.entries(sources)) {
  const folder = path.slice(path.indexOf(ROOT) + ROOT.length, path.lastIndexOf('/'))
  for (const m of source.matchAll(BINDING)) {
    methods.set(Number(m[3]), { name: `${folder}.${m[1]}`, result: m[2] ?? '' })
  }
}

export const methodOf = (id: number): Method | undefined => methods.get(id)

const ids = new Map([...methods].map(([id, m]) => [m.name, id]))

/** A bound method's id by its folder and name, as `settings.PrefSpecs`. */
export const idOf = (name: string): number => ids.get(name) ?? 0

/** An empty value of a declared result type: what a backend with nothing to report answers. */
export function emptyOf(result: string): unknown {
  if (result.endsWith('| null') || result === 'void') {
    return null
  }
  if (result.endsWith('[]')) {
    return []
  }
  if (result === 'string') {
    return ''
  }
  if (result === 'boolean') {
    return false
  }
  if (result === 'number') {
    return 0
  }
  return {}
}
