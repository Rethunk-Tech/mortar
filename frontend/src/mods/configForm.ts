type ConfigNode =
  | { kind: 'bool'; value: boolean }
  | { kind: 'int'; value: string }
  | { kind: 'float'; value: string }
  | { kind: 'string'; value: string }
  | { kind: 'strings'; value: string[] }
  | { kind: 'object'; entries: { key: string; node: ConfigNode }[] }
  | { kind: 'readonly'; json: string }

class Parser {
  private at = 0
  constructor(private readonly text: string) {}
  value(): ConfigNode {
    this.space()
    const c = this.text[this.at]
    if (c === '{') return this.object()
    if (c === '[') return this.array()
    if (c === '"') return { kind: 'string', value: this.string() }
    if (this.text.startsWith('true', this.at)) {
      this.at += 4
      return { kind: 'bool', value: true }
    }
    if (this.text.startsWith('false', this.at)) {
      this.at += 5
      return { kind: 'bool', value: false }
    }
    if (this.text.startsWith('null', this.at)) {
      this.at += 4
      return { kind: 'readonly', json: 'null' }
    }
    const match = this.text.slice(this.at).match(/^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/)
    if (!match) throw new Error(`Invalid config near ${this.at}`)
    this.at += match[0].length
    return { kind: /[.eE]/.test(match[0]) ? 'float' : 'int', value: match[0] }
  }
  private object(): ConfigNode {
    this.at++
    const entries: { key: string; node: ConfigNode }[] = []
    this.space()
    while (this.text[this.at] !== '}') {
      const key = this.string()
      this.space()
      this.at++
      entries.push({ key, node: this.value() })
      this.space()
      if (this.text[this.at] === ',') {
        this.at++
        this.space()
      }
    }
    this.at++
    return { kind: 'object', entries }
  }
  private array(): ConfigNode {
    this.at++
    const values: ConfigNode[] = []
    this.space()
    while (this.text[this.at] !== ']') {
      values.push(this.value())
      this.space()
      if (this.text[this.at] === ',') {
        this.at++
        this.space()
      }
    }
    this.at++
    return { kind: 'readonly', json: `[${values.map((v) => serialize(v)).join(',')}]` }
  }
  private string(): string {
    const start = this.at++
    while (this.at < this.text.length) {
      if (this.text[this.at] === '\\') this.at += 2
      else if (this.text[this.at++] === '"')
        return JSON.parse(this.text.slice(start, this.at)) as string
    }
    throw new Error('Unterminated config string')
  }
  private space() {
    while (/\s/.test(this.text[this.at] ?? '')) this.at++
  }
}

function fromValue(value: unknown): ConfigNode {
  if (value === null || typeof value === 'undefined') {
    return { kind: 'readonly', json: 'null' }
  }
  if (typeof value === 'boolean') {
    return { kind: 'bool', value }
  }
  if (typeof value === 'number' && Number.isFinite(value)) {
    return Number.isInteger(value)
      ? { kind: 'int', value: String(value) }
      : { kind: 'float', value: String(value) }
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

function serialize(node: ConfigNode): string {
  switch (node.kind) {
    case 'bool':
      return node.value ? 'true' : 'false'
    case 'int':
    case 'float':
      return node.value
    case 'string':
      return JSON.stringify(node.value)
    case 'strings':
      return `[${node.value.map((v) => JSON.stringify(v)).join(',')}]`
    case 'object':
      return `{${node.entries.map((e) => `${JSON.stringify(e.key)}:${serialize(e.node)}`).join(',')}}`
    case 'readonly':
      return node.json
    default: {
      const exhaustive: never = node
      return String(exhaustive)
    }
  }
}

export type { ConfigNode }

export function parseConfig(text: string): ConfigNode {
  return new Parser(text).value()
}

export function stringifyConfig(node: ConfigNode): string {
  return `${serialize(node)}\n`
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
