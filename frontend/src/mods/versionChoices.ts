import type { Want } from '../queue/actions.ts'
import { isSameVersion } from './nexusFormat.ts'

/** One version a source lists for a mod; fileId is the Nexus file of that version. */
export interface ListedVersion {
  version: string
  fileId?: number
  /** What tells same-numbered files apart, such as a Nexus file's title. */
  note?: string
}

export interface VersionChoice {
  key: string
  label: string
  version: string
  want: Want
}

export interface VersionTarget {
  source: string
  id: string
}

const RELEASE_SOURCES = new Set([
  'nexus',
  'github',
  'thunderstore',
  'modrinth',
  'curseforge',
  'vsmoddb',
  'paradox',
])

/** Whether the source lists a mod's versions (itch.io does not). */
export const listsVersions = (source: string) => RELEASE_SOURCES.has(source)

/** The mod an entry's source names on its site, or null for a mod no site can list versions of. */
export function targetOfSource(src: {
  kind: string
  name: string
  modId?: number | undefined
  repo?: string | undefined
}): VersionTarget | null {
  if (!listsVersions(src.kind)) {
    return null
  }
  switch (src.kind) {
    case 'nexus':
      return src.modId ? { source: 'nexus', id: String(src.modId) } : null
    case 'github':
      return src.repo ? { source: 'github', id: src.repo } : null
    default:
      return src.name ? { source: src.kind, id: src.name } : null
  }
}

/** What to queue to install a listed version; with replaces, the entry it takes the place of (even when older). */
function wantOf(target: VersionTarget, v: ListedVersion, replaces: string): Want {
  const how: Want = replaces
    ? { kind: 'update', currentKey: replaces, exact: true }
    : { kind: 'install' }
  switch (target.source) {
    case 'nexus':
      return { ...how, modId: Number(target.id), fileId: v.fileId ?? 0, version: v.version }
    case 'github':
      return { ...how, repo: target.id, tag: v.version, version: v.version }
    case 'thunderstore':
      return { ...how, package: target.id, version: v.version }
    default:
      return { ...how, source: target.source, package: target.id, version: v.version }
  }
}

/** The choices a picker lists, newest first as the site gives them; installed marks the one the profile holds. */
export function versionChoices(
  target: VersionTarget,
  listed: ListedVersion[],
  replaces: string,
): VersionChoice[] {
  return listed.map((v, i) => ({
    key: `${i}:${v.version}:${v.fileId ?? 0}`,
    label: v.note && v.note !== v.version ? `${v.version} · ${v.note}` : v.version,
    version: v.version,
    want: wantOf(target, v, replaces),
  }))
}

export const isInstalledVersion = (choice: VersionChoice, installed: string) =>
  installed !== '' && isSameVersion(choice.version, installed)
