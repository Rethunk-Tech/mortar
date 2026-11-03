import type {
  Entry,
  Profile,
  Source,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { sameId } from '../mods/lookup.ts'
import { nexusDomain } from '../mods/nexusDomain.ts'
import { nexusModUrl } from '../mods/nexusUrl.ts'

const DISCORD_LIMIT = 2000
const BUNDLED = new Set(['smapi', 'mortar'])

type ModListFormat = 'markdown' | 'plain' | 'discord' | 'thunderstore'

interface ModListItem {
  name: string
  version: string
  url: string
  enabled: boolean
  /** The Thunderstore dependency string (Namespace-Name-1.2.3), when the mod came from Thunderstore. */
  dependency?: string
}

interface GroupLabels {
  enabled: string
  disabled: string
}

const defaultLabels: GroupLabels = { enabled: 'Enabled', disabled: 'Disabled' }

function pageUrl(source: Source): string {
  if (source.kind === 'nexus' && source.modId) {
    return nexusModUrl(source.modId, nexusDomain())
  }
  if (source.kind === 'github' && source.repo) {
    return `https://github.com/${source.repo}`
  }
  return ''
}

// Thunderstore names a package Namespace-Name and a dependency Namespace-Name-Version.
function dependencyOf(source: Source): string | undefined {
  return source.kind === 'thunderstore' && source.name && source.version
    ? `${source.name}-${source.version}`
    : undefined
}

function isOn(entry: Entry, id: string): boolean {
  return !(entry.disabled ?? []).some((disabled) => sameId(disabled, id))
}

function markdownLine(item: ModListItem): string {
  const name = escapeMarkdown(item.name)
  const body = item.url ? `[${name}](${item.url})` : name
  return item.version ? `- ${body} ${item.version}` : `- ${body}`
}

function plainLine(item: ModListItem): string {
  const nameVer = item.version ? `${item.name} v${item.version}` : item.name
  return item.url ? `${nameVer} - ${item.url}` : nameVer
}

function grouped(
  items: readonly ModListItem[],
  line: (item: ModListItem) => string,
  labels: GroupLabels,
): string {
  if (!items.some((item) => !item.enabled)) {
    return items.map(line).join('\n')
  }
  const on = items.filter((item) => item.enabled)
  const off = items.filter((item) => !item.enabled)
  const parts: string[] = []
  if (on.length > 0) {
    parts.push(labels.enabled, ...on.map(line))
  }
  if (off.length > 0) {
    if (parts.length > 0) {
      parts.push('')
    }
    parts.push(labels.disabled, ...off.map(line))
  }
  return parts.join('\n')
}

function sliceLine(line: string, chunks: string[]): string {
  let rest = line
  while (rest.length > DISCORD_LIMIT) {
    chunks.push(rest.slice(0, DISCORD_LIMIT))
    rest = rest.slice(DISCORD_LIMIT)
  }
  return rest
}

function splitDiscord(text: string): string[] {
  if (text.length <= DISCORD_LIMIT) {
    return text === '' ? [''] : [text]
  }
  const chunks: string[] = []
  let current = ''
  for (const line of text.split('\n')) {
    const next = current === '' ? line : `${current}\n${line}`
    if (next.length <= DISCORD_LIMIT) {
      current = next
    } else {
      if (current !== '') {
        chunks.push(current)
      }
      current = line.length <= DISCORD_LIMIT ? line : sliceLine(line, chunks)
    }
  }
  if (current !== '') {
    chunks.push(current)
  }
  return chunks
}

function numbered(parts: string[]): DiscordPart[] {
  return parts.map((text, n) => ({ id: `discord-${n + 1}-${text.length}`, n: n + 1, text }))
}

// Several parts of one download can share a name and version; the list names them once.
function distinct(items: readonly ModListItem[]): ModListItem[] {
  const seen = new Set<string>()
  return items.filter((item) => {
    const key = JSON.stringify([item.name, item.version, item.url, item.enabled])
    return !seen.has(key) && Boolean(seen.add(key))
  })
}

function withDependency(source: Source): { dependency?: string } {
  const dependency = dependencyOf(source)
  return dependency ? { dependency } : {}
}

function collectItems(profile: Profile | undefined, keys: readonly string[]): ModListItem[] {
  if (!profile) {
    return []
  }
  const only = keys.length > 0 ? new Set(keys) : null
  const items: ModListItem[] = []
  for (const entry of profile.entries ?? []) {
    const skip = BUNDLED.has(entry.source.kind) || (only !== null && !only.has(entry.key))
    if (!skip) {
      const mods = entry.mods ?? []
      if (mods.length === 0) {
        items.push({
          name: entry.source.name || entry.key,
          version: entry.source.version ?? '',
          url: pageUrl(entry.source),
          enabled: true,
          ...withDependency(entry.source),
        })
      } else {
        for (const mod of mods) {
          items.push({
            name: mod.name,
            version: mod.version,
            url: pageUrl(entry.source),
            enabled: isOn(entry, mod.id),
            ...withDependency(entry.source),
          })
        }
      }
    }
  }
  return items
}

// Bundled SMAPI mods and the Mortar bridge are left out, matching what a share carries.
function listItems(profile: Profile | undefined, keys: readonly string[] = []): ModListItem[] {
  return distinct(collectItems(profile, keys))
}

/** How many mods the list names once because another has the same name, version, link and state. */
function repeatedItems(profile: Profile | undefined, keys: readonly string[] = []): number {
  return collectItems(profile, keys).length - listItems(profile, keys).length
}

// Names sit in Markdown link text, so the characters that would break a link or emphasis are escaped.
function escapeMarkdown(name: string): string {
  return name.replace(/([\\`*_[\]()])/g, '\\$1')
}

function formatMarkdown(
  items: readonly ModListItem[],
  labels: GroupLabels = defaultLabels,
): string {
  return grouped(items, markdownLine, labels)
}

function formatPlain(items: readonly ModListItem[], labels: GroupLabels = defaultLabels): string {
  return grouped(items, plainLine, labels)
}

function asDiscord(
  items: readonly ModListItem[],
  labels: GroupLabels = defaultLabels,
): DiscordPart[] {
  return numbered(splitDiscord(formatMarkdown(items, labels)))
}

/** The switched-on Thunderstore mods as dependency strings, one a line, as a package manifest lists them. */
function formatDependencies(items: readonly ModListItem[]): string {
  return [...new Set(items.filter((i) => i.enabled && i.dependency).map((i) => i.dependency))].join(
    '\n',
  )
}

/** The mods a dependency list leaves out: those not on Thunderstore and those switched off. */
function skippedForDependencies(items: readonly ModListItem[]): string[] {
  return items.filter((i) => !(i.enabled && i.dependency)).map((i) => i.name)
}

function formatModList(
  format: ModListFormat,
  items: readonly ModListItem[],
  labels: GroupLabels = defaultLabels,
): DiscordPart[] {
  if (format === 'plain') {
    return numbered([formatPlain(items, labels)])
  }
  if (format === 'thunderstore') {
    return numbered([formatDependencies(items)])
  }
  if (format === 'discord') {
    return asDiscord(items, labels)
  }
  return numbered([formatMarkdown(items, labels)])
}

interface DiscordPart {
  id: string
  n: number
  text: string
}

export type { ModListFormat, ModListItem }
export {
  DISCORD_LIMIT,
  escapeMarkdown,
  formatMarkdown,
  formatModList,
  formatPlain,
  listItems,
  repeatedItems,
  skippedForDependencies,
  splitDiscord,
}
