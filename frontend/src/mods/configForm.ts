type ConfigNode =
  | { kind: 'bool'; value: boolean }
  | { kind: 'int'; value: number }
  | { kind: 'float'; value: number }
  | { kind: 'string'; value: string }
  | { kind: 'strings'; value: string[] }
  | { kind: 'object'; entries: { key: string; node: ConfigNode }[] }
  | { kind: 'readonly'; json: string }

function fromValue(value: unknown): ConfigNode {
  if (value === null || typeof value === 'undefined') {
    return { kind: 'readonly', json: 'null' }
  }
  if (typeof value === 'boolean') {
    return { kind: 'bool', value }
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return Number.isInteger(value) ? { kind: 'int', value } : { kind: 'float', value }
  }
  if (typeof value === 'string') {
    return { kind: 'string', value }
  }
  if (Array.isArray(value)) {
    if (value.every((item) => typeof item === 'string')) {
      return { kind: 'strings', value: value as string[] }
    }
    return { kind: 'readonly', json: JSON.stringify(value) }
  }
  if (typeof value === 'object') {
    return {
      kind: 'object',
      entries: Object.entries(value as Record<string, unknown>).map(([key, child]) => ({
        key,
        node: fromValue(child),
      })),
    }
  }
  return { kind: 'readonly', json: JSON.stringify(value) }
}

function toValue(node: ConfigNode): unknown {
  switch (node.kind) {
    case 'bool':
    case 'int':
    case 'float':
    case 'string':
    case 'strings':
      return node.value
    case 'object':
      return Object.fromEntries(node.entries.map((e) => [e.key, toValue(e.node)]))
    case 'readonly':
      return JSON.parse(node.json) as unknown
    default: {
      const exhaustive: never = node
      return exhaustive
    }
  }
}

export type { ConfigNode }

export function parseConfig(text: string): ConfigNode {
  return fromValue(JSON.parse(text) as unknown)
}

export function stringifyConfig(node: ConfigNode): string {
  return `${JSON.stringify(toValue(node), null, 2)}\n`
}

export function setAt(node: ConfigNode, path: readonly number[], next: ConfigNode): ConfigNode {
  if (path.length === 0) {
    return next
  }
  if (node.kind !== 'object') {
    return node
  }
  const [head, ...rest] = path
  return {
    kind: 'object',
    entries: node.entries.map((entry, i) =>
      i === head ? { key: entry.key, node: setAt(entry.node, rest, next) } : entry,
    ),
  }
}
