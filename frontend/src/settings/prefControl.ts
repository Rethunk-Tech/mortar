type PrefControl = 'switch' | 'select' | 'number' | 'text'

type ChoiceStyle = 'segmented' | 'cards' | 'select'

const SEGMENT_MAX = 4
const SEGMENT_LABEL_CHARS = 48

function prefControl(type: string): PrefControl {
  if (type === 'bool') {
    return 'switch'
  }
  if (type === 'enum') {
    return 'select'
  }
  if (type === 'int') {
    return 'number'
  }
  return 'text'
}

// Options that explain themselves become cards; a few short options become a button strip; the rest stay a list.
function choiceStyle(options: { label: string; hint?: string }[]): ChoiceStyle {
  if (options.length > 1 && options.every((o) => o.hint)) {
    return 'cards'
  }
  const chars = options.reduce((n, o) => n + o.label.length, 0)
  return options.length > 1 && options.length <= SEGMENT_MAX && chars <= SEGMENT_LABEL_CHARS
    ? 'segmented'
    : 'select'
}

export { choiceStyle, prefControl }
