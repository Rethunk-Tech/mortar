interface ModListEntry {
  enabled: boolean
  name: string
  version: string
  nexusUrl?: string
}

type ModListFormat = 'markdown' | 'plain'

const enabledOf = (mods: readonly ModListEntry[]) => mods.filter((m) => m.enabled)

const labeled = (mod: ModListEntry) => {
  const name = mod.name.trim()
  const version = mod.version.trim()
  if (name === '') {
    return version
  }
  return version === '' ? name : `${name} ${version}`
}

const line = (mod: ModListEntry, format: ModListFormat) => {
  const text = labeled(mod)
  if (format === 'markdown') {
    return mod.nexusUrl ? `- [${text}](${mod.nexusUrl})` : `- ${text}`
  }
  return mod.nexusUrl ? `${text} ${mod.nexusUrl}` : text
}

const formatModList = (mods: readonly ModListEntry[], format: ModListFormat) =>
  enabledOf(mods)
    .map((mod) => line(mod, format))
    .join('\n')

export type { ModListFormat }
export { formatModList }
