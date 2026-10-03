export type MenuControl =
  | 'switch'
  | 'number'
  | 'text'
  | 'select'
  | 'keybind'
  | 'color'
  | 'image'
  | 'section'
  | 'subHeader'
  | 'paragraph'
  | 'pageLink'
  | 'readonly'

export function menuControl(kind: string): MenuControl {
  switch (kind) {
    case 'bool':
      return 'switch'
    case 'int':
    case 'float':
      return 'number'
    case 'text':
      return 'text'
    case 'choice':
      return 'select'
    case 'keybind':
    case 'keybindList':
      return 'keybind'
    case 'color':
      return 'color'
    case 'image':
      return 'image'
    case 'sectionTitle':
      return 'section'
    case 'subHeader':
      return 'subHeader'
    case 'paragraph':
      return 'paragraph'
    case 'pageLink':
      return 'pageLink'
    case 'image display':
    case 'imageDisplay':
    case 'complex':
      return 'readonly'
    default:
      return 'readonly'
  }
}

export function editableKind(kind: string): boolean {
  switch (menuControl(kind)) {
    case 'section':
    case 'subHeader':
    case 'paragraph':
    case 'pageLink':
    case 'readonly':
      return false
    default:
      return true
  }
}
