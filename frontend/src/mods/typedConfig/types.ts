type ConfigValue = boolean | number | string | string[]

type EntryType = 'bool' | 'int' | 'float' | 'enum' | 'string' | 'color' | 'list'

interface ConfigEntry {
  key: string
  // A friendlier name when the mod gave one; the key is still what is written.
  label?: string
  type: EntryType
  default: ConfigValue
  value: ConfigValue
  description: string
  min?: number
  max?: number
  options?: string[]
}

interface ConfigSection {
  name: string
  entries: ConfigEntry[]
}

interface ConfigFile {
  name: string
  label: string
  // Read when the file is first shown.
  sections: ConfigSection[]
}

export type { ConfigEntry, ConfigFile, ConfigSection, ConfigValue, EntryType }
