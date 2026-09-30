import type {
  Entry,
  Profile,
  Source,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'

const LIMIT = 2000
const BUNDLED = new Set(['smapi', 'mortar'])
const NEXUS_PAGE = 'https://www.nexusmods.com/stardewvalley/mods/'

type Format = 'markdown' | 'plain' | 'discord'

interface Item {
  name: string
  version: string
  url: string
  enabled: boolean
}

interface GroupLabels {
  enabled: string
  disabled: string
}

const defaultLabels: GroupLabels = { enabled: 'Enabled', disabled: 'Disabled' }

function pageUrl(source: Source): string {
  if (source.kind === 'nexus' && source.modId) {
    return `${NEXUS_PAGE}${source.modId}`
  }
  if (source.kind === 'github' && source.repo) {
    return `https://github.com/${source.repo}`
  }
  return ''
}

function isOn(entry: Entry, uniqueId: string): boolean {
  return !(entry.disabled ?? []).some((id) => id.toLowerCase() === uniqueId.toLowerCase())
}

function markdownLine(item: Item): string {
  const name = escapeName(item.name)
  const body = item.url ? `[${name}](${item.url})` : name
  return item.version ? `- ${body} ${item.version}` : `- ${body}`
}

function plainLine(item: Item): string {
  const nameVer = item.version ? `${item.name} v${item.version}` : item.name
  return item.url ? `${nameVer} - ${item.url}` : nameVer
}

function grouped(
  items: readonly Item[],
  line: (item: Item) => string,
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
  while (rest.length > LIMIT) {
    chunks.push(rest.slice(0, LIMIT))
    rest = rest.slice(LIMIT)
  }
  return rest
}

function splitText(text: string): string[] {
  if (text.length <= LIMIT) {
    return text === '' ? [''] : [text]
  }
  const chunks: string[] = []
  let current = ''
  for (const line of text.split('\n')) {
    const next = current === '' ? line : `${current}\n${line}`
    if (next.length <= LIMIT) {
      current = next
    } else {
      if (current !== '') {
        chunks.push(current)
      }
      current = line.length <= LIMIT ? line : sliceLine(line, chunks)
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

// Bundled SMAPI mods and the Mortar bridge are left out, matching what a share carries.
function itemsOf(profile: Profile | undefined, keys: readonly string[] = []): Item[] {
  if (!profile) {
    return []
  }
  const only = keys.length > 0 ? new Set(keys) : null
  const items: Item[] = []
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
        })
      } else {
        for (const mod of mods) {
          items.push({
            name: mod.name,
            version: mod.version,
            url: pageUrl(entry.source),
            enabled: isOn(entry, mod.uniqueId),
          })
        }
      }
    }
  }
  return items
}

// Names sit in Markdown link text, so the characters that would break a link or emphasis are escaped.
function escapeName(name: string): string {
  return name.replace(/([\\`*_[\]()])/g, '\\$1')
}

function asMarkdown(items: readonly Item[], labels: GroupLabels = defaultLabels): string {
  return grouped(items, markdownLine, labels)
}

function asPlain(items: readonly Item[], labels: GroupLabels = defaultLabels): string {
  return grouped(items, plainLine, labels)
}

function asDiscord(items: readonly Item[], labels: GroupLabels = defaultLabels): DiscordPart[] {
  return numbered(splitText(asMarkdown(items, labels)))
}

function formatted(
  format: Format,
  items: readonly Item[],
  labels: GroupLabels = defaultLabels,
): DiscordPart[] {
  if (format === 'plain') {
    return numbered([asPlain(items, labels)])
  }
  if (format === 'discord') {
    return asDiscord(items, labels)
  }
  return numbered([asMarkdown(items, labels)])
}

interface DiscordPart {
  id: string
  n: number
  text: string
}

export type ModListFormat = Format
export type ModListItem = Item
export const DISCORD_LIMIT = LIMIT
export const listItems = itemsOf
export const escapeMarkdown = escapeName
export const formatMarkdown = asMarkdown
export const formatPlain = asPlain
export const splitDiscord = splitText
export const formatDiscord = (items: readonly Item[], labels?: GroupLabels) =>
  asDiscord(items, labels).map((p) => p.text)
export const formatModList = formatted
