import { useLoaded } from '@rethunk/react-load-hooks'
import { useEffect } from 'react'
import { Details as ReadDetails } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import { ReleaseChangelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { loadDetails, useNexusEntry } from './nexusDetails.ts'
import type { ListedVersion } from './versionChoices.ts'

const MS_PER_S = 1000

// A release time only when the site gave a readable one.
const dated = (ms: number) => (Number.isFinite(ms) && ms > 0 ? { released: ms } : {})

export type Listed =
  | { state: 'loading' }
  | { state: 'error'; error: InlineError; retry: () => void }
  | { state: 'ready'; versions: ListedVersion[] }

// A Nexus mod's files come from the details the Mods tab reads too; archived files are not offered. Every other source
// lists the versions it publishes, newest first.
function useNexusVersions(game: string, modId: number): Listed {
  const entry = useNexusEntry(game, modId)
  const wanted = modId > 0
  useEffect(() => {
    if (wanted) {
      loadDetails(game, modId).catch(() => undefined)
    }
  }, [wanted, game, modId])
  const files = entry?.details?.files
  if (files) {
    return {
      state: 'ready',
      versions: files
        .filter((f) => f.category !== 'ARCHIVED')
        .toSorted((a, b) => Date.parse(b.uploaded) - Date.parse(a.uploaded))
        .map((f) => ({
          version: f.version || f.name,
          fileId: f.fileId,
          note: f.name,
          ...dated(Date.parse(f.uploaded)),
        })),
    }
  }
  if (entry?.error) {
    const error = { message: entry.error, details: entry.error }
    return { state: 'error', error, retry: () => loadDetails(game, modId).then(() => undefined) }
  }
  return { state: 'loading' }
}

function useSiteVersions(game: string, source: string, id: string): Listed {
  const { data, error, reload } = useLoaded<ListedVersion[] | null>(
    source === 'nexus'
      ? null
      : () =>
          source === 'github'
            ? ReleaseChangelog(id).then((logs) =>
                (logs ?? []).map((l) => ({
                  version: l.version,
                  ...dated(Date.parse(l.date ?? '')),
                })),
              )
            : ReadDetails(game, source, id).then((d) =>
                (d.versions ?? []).map((version, i) => ({
                  version,
                  ...dated((d.released?.[i] ?? 0) * MS_PER_S),
                })),
              ),
    [game, source, id],
    null,
  )
  if (error !== null) {
    return { state: 'error', error: inlineError(error), retry: reload }
  }
  return data === null ? { state: 'loading' } : { state: 'ready', versions: data }
}

/** The versions a source lists for one mod. */
export function useListedVersions(game: string, source: string, id: string): Listed {
  const nexus = useNexusVersions(game, source === 'nexus' ? Number(id) : 0)
  const site = useSiteVersions(game, source, id)
  return source === 'nexus' ? nexus : site
}
