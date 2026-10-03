interface Hint {
  description: string
  defaultValue: string
  section: string
}

type ConfigNode =
  | { kind: 'bool'; value: boolean; hint?: Hint }
  | { kind: 'int'; value: string; hint?: Hint }
  | { kind: 'float'; value: string; hint?: Hint }
  | { kind: 'string'; value: string; hint?: Hint }
  | { kind: 'keybind'; value: string; hint?: Hint }
  | {
      kind: 'choice'
      value: string
      options: readonly string[]
      multiple: boolean
      allowBlank: boolean
      defaultValue: string
      description: string
      section: string
    }
  | { kind: 'list'; items: ConfigNode[] }
  | { kind: 'object'; entries: { key: string; node: ConfigNode }[] }
  | { kind: 'readonly'; json: string }

const numberPattern = /^-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?/
const floatPattern = /[.eE]/
const whitespacePattern = /\s/
const namedButton =
  /^(?:None|MouseLeft|MouseRight|MouseMiddle|MouseX1|MouseX2|LeftShift|RightShift|LeftControl|RightControl|LeftAlt|RightAlt|LeftWindows|RightWindows|Enter|Space|Escape|Tab|Back|CapsLock|Up|Down|Left|Right|Home|End|PageUp|PageDown|Insert|Delete|Pause|PrintScreen|VolumeUp|VolumeDown|VolumeMute|MediaNextTrack|MediaPreviousTrack|MediaStop|MediaPlayPause|Oemtilde|OemMinus|OemPlus|OemOpenBrackets|OemCloseBrackets|OemPipe|OemSemicolon|OemQuotes|OemComma|OemPeriod|OemQuestion|OemBackslash|ControllerA|ControllerB|ControllerX|ControllerY|ControllerBack|ControllerStart|LeftStick|RightStick|LeftShoulder|RightShoulder|LeftTrigger|RightTrigger|DPadUp|DPadDown|DPadLeft|DPadRight|BigButton)$/
const letterButton = /^[A-Z]$/
const digitButton = /^D[0-9]$/
const functionButton = /^F([1-9]|1[0-9]|2[0-4])$/
const numpadDigit = /^NumPad[0-9]$/

function parseOneKeybind(text: string): boolean {
  const parts = text.split('+').map((p) => p.trim())
  return parts.length > 0 && parts.every((p) => p.length > 0 && isSButton(p))
}

function isSButton(name: string): boolean {
  return (
    namedButton.test(name) ||
    letterButton.test(name) ||
    digitButton.test(name) ||
    functionButton.test(name) ||
    numpadDigit.test(name) ||
    name === 'NumPadEnter' ||
    name === 'NumPadMinus' ||
    name === 'NumPadPlus' ||
    name === 'NumPadMultiply' ||
    name === 'NumPadDivide' ||
    name === 'NumPadDecimal'
  )
}

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
      const value = this.string()
      return isKeybind(value) ? { kind: 'keybind', value } : { kind: 'string', value }
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
    // A list editor adds items of one kind, so only single-kind scalar arrays are editable as a list.
    if (values.every(isScalar) && new Set(values.map((v) => v.kind)).size <= 1) {
      return { kind: 'list', items: values }
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

function isScalar(node: ConfigNode): boolean {
  return (
    node.kind === 'bool' ||
    node.kind === 'int' ||
    node.kind === 'float' ||
    node.kind === 'string' ||
    node.kind === 'keybind'
  )
}

function serialize(node: ConfigNode): string {
  switch (node.kind) {
    case 'bool':
      return node.value ? 'true' : 'false'
    case 'int':
    case 'float':
      return node.value
    case 'string':
    case 'keybind':
    case 'choice':
      return JSON.stringify(node.value)
    case 'list':
      return `[${node.items.map((v) => serialize(v)).join(',')}]`
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

function isKeybind(value: string): boolean {
  if (!value.trim()) {
    return false
  }
  return value
    .split(',')
    .map((p) => p.trim())
    .every((p) => p.length > 0 && parseOneKeybind(p))
}

function parseConfig(text: string): ConfigNode {
  return new Parser(text).value()
}

function stringifyConfig(node: ConfigNode): string {
  return `${serialize(node)}\n`
}

function setAt(node: ConfigNode, path: readonly number[], next: ConfigNode): ConfigNode {
  if (path.length === 0) {
    return next
  }
  const [head, ...rest] = path
  if (node.kind === 'object') {
    return {
      kind: 'object',
      entries: node.entries.map((entry, i) =>
        i === head ? { key: entry.key, node: setAt(entry.node, rest, next) } : entry,
      ),
    }
  }
  if (node.kind === 'list') {
    return {
      kind: 'list',
      items: node.items.map((item, i) => (i === head ? setAt(item, rest, next) : item)),
    }
  }
  return node
}

function emptyItem(items: readonly ConfigNode[]): ConfigNode {
  const [sample] = items
  switch (sample?.kind) {
    case 'bool':
      return { kind: 'bool', value: false }
    case 'int':
      return { kind: 'int', value: '0' }
    case 'float':
      return { kind: 'float', value: '0' }
    case 'keybind':
      return { kind: 'keybind', value: 'None' }
    default:
      return { kind: 'string', value: '' }
  }
}

export type { ConfigNode, Hint }
export { emptyItem, isKeybind, parseConfig, setAt, stringifyConfig }
