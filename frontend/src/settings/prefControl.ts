export type PrefControl = 'switch' | 'select' | 'number' | 'text'

export function prefControl(type: string): PrefControl {
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
