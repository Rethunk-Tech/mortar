type ConfigValue = boolean | number | string | string[]

type EntryType = 'bool' | 'int' | 'float' | 'enum' | 'string' | 'color' | 'list' | 'multi'

interface ConfigEntry {
  key: string
  // A friendlier name when the mod gave one; the key is still what is written.
  label?: string
  // 'multi' is several of `options` at once, held as one comma-joined string.
  type: EntryType
  default: ConfigValue
  value: ConfigValue
  description: string
  min?: number
  max?: number
  options?: string[]
  // How the mod names each option, when that differs from the value written.
  optionLabels?: string[]
  // The value waits for the game's next start; the default is the value the game has now.
  pending?: boolean
  // Only the game can change it.
  readOnly?: boolean
  // What the game reported when it could not apply the last change.
  note?: string
}

interface ConfigSection {
  name: string
  entries: ConfigEntry[]
}

interface ConfigFile {
  name: string
  label: string
  // 'gmcm' is the mod's in-game menu, whose edits wait for the next start.
  format: string
  // Read when the file is first shown.
  sections: ConfigSection[]
}

export type { ConfigEntry, ConfigFile, ConfigSection, ConfigValue, EntryType }
