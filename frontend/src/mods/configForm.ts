type ConfigNode =
  | { kind: 'bool'; value: boolean }
  | { kind: 'int'; value: string }
  | { kind: 'float'; value: string }
  | { kind: 'string'; value: string }
  | { kind: 'strings'; value: string[] }
  | { kind: 'object'; entries: { key: string; node: ConfigNode }[] }
  | { kind: 'readonly'; json: string }

const numberPattern = /^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/
const floatPattern = /[.eE]/
const whitespacePattern = /\s/

class Parser {
  private at = 0
  private readonly text: string
  constructor(text: string) {
    this.text = text
  }
  value(): ConfigNode {
    this.space()
    const c = this.text[this.at]
    if (c === '{') {
      return this.object()
    }
    if (c === '[') {
      return this.array()
    }
    if (c === '"') {
      return { kind: 'string', value: this.string() }
    }
    if (this.text.startsWith('true', this.at)) {
      this.at += 'true'.length
      return { kind: 'bool', value: true }
    }
    if (this.text.startsWith('false', this.at)) {
      this.at += 'false'.length
      return { kind: 'bool', value: false }
    }
    if (this.text.startsWith('null', this.at)) {
      this.at += 'null'.length
      return { kind: 'readonly', json: 'null' }
    }
    const match = this.text.slice(this.at).match(numberPattern)
    if (!match) {
      throw new Error(`Invalid config near ${this.at}`)
    }
    this.at += match[0].length
    return { kind: floatPattern.test(match[0]) ? 'float' : 'int', value: match[0] }
  }
  private object(): ConfigNode {
    this.at += 1
    const entries: { key: string; node: ConfigNode }[] = []
    this.space()
    while (this.text[this.at] !== '}') {
      const key = this.string()
      this.space()
      this.at += 1
      entries.push({ key, node: this.value() })
      this.space()
      if (this.text[this.at] === ',') {
        this.at += 1
        this.space()
      }
    }
    this.at += 1
    return { kind: 'object', entries }
  }
  private array(): ConfigNode {
    this.at += 1
    const values: ConfigNode[] = []
    this.space()
    while (this.text[this.at] !== ']') {
      values.push(this.value())
      this.space()
      if (this.text[this.at] === ',') {
        this.at += 1
        this.space()
      }
    }
    this.at += 1
    if (values.every((v): v is Extract<ConfigNode, { kind: 'string' }> => v.kind === 'string')) {
      return { kind: 'strings', value: values.map((v) => v.value) }
    }
    return { kind: 'readonly', json: `[${values.map((v) => serialize(v)).join(',')}]` }
  }
  private string(): string {
    const start = this.at
    this.at += 1
    while (this.at < this.text.length) {
      if (this.text[this.at] === '\\') {
        this.at += 2
      } else if (this.text[this.at] === '"') {
        this.at += 1
        return JSON.parse(this.text.slice(start, this.at)) as string
      } else {
        this.at += 1
      }
    }
    throw new Error('Unterminated config string')
  }
  private space() {
    while (whitespacePattern.test(this.text[this.at] ?? '')) {
      this.at += 1
    }
  }
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
