import { sameValue } from '@rethunk/react-load-hooks'
import { useEffect } from 'react'
import { create } from 'zustand'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/models.ts'
import type {
  Entry,
  ModFact,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ModFacts } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { ListRow } from './listColumns.ts'
import { useUpdates } from './updates.ts'

/** What a mod's own source says about it, whichever source that is. */
export interface Figures {
  latest: string
  /** When the installed version was published. */
  released: string
  updated: string
  rating: number
  downloads: number
}

type FactsById = Readonly<Record<string, ModFact | undefined>>

const NONE: FactsById = {}

export const useModFacts = create<{ key: string; checked: unknown; byId: FactsById }>(() => ({
  key: '',
  checked: null,
  byId: NONE,
}))

/** next, with each fact that reads the same as in prev kept as prev's object (and prev itself when nothing changed), so
 * a re-read after an update check redraws only the rows whose facts did. */
export function keepSameFacts(prev: FactsById, next: FactsById): FactsById {
  let changed = Object.keys(prev).length !== Object.keys(next).length
  const kept: Record<string, ModFact | undefined> = {}
  for (const [id, fact] of Object.entries(next)) {
    const old = prev[id]
    const same = old !== undefined && sameValue(old, fact)
    kept[id] = same ? old : fact
    changed ||= !same
  }
  return changed ? kept : prev
}

/** The open profile's facts, read again only when its set of packages changes or an update check ran: a switch or a
 * note changes none of them, reading them can cost the backend half a second, and the check is what refreshes the
 * source listings they come from. */
export function useProfileModFacts(game: string, profile: Profile): FactsById {
  const key = `${game}\n${profile.id}\n${(profile.entries ?? []).map((e) => e.key).join('\n')}`
  const checked = useUpdates((s) => s.updates)
  useEffect(() => {
    const now = useModFacts.getState()
    if (now.key === key && now.checked === checked) {
      return
    }
    let live = true
    ModFacts(game, profile.id)
      .then((byId) => {
        if (live) {
          useModFacts.setState((s) => ({ key, checked, byId: keepSameFacts(s.byId, byId ?? NONE) }))
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [key, checked, game, profile.id])
  return useModFacts((s) => s.byId)
}

/** A row's facts and figures, each present only when known. */
export function figureFields(
  entry: Entry | undefined,
  details: Details | undefined,
  fact: ModFact | undefined,
): Pick<ListRow, 'facts' | 'figures'> {
  const figures = figuresOf(entry, details, fact)
  return { ...(fact ? { facts: fact } : {}), ...(figures ? { figures } : {}) }
}

/** A Nexus mod's figures come from the page details Mortar caches; every other source's from its facts. */
function figuresOf(
  entry: Entry | undefined,
  details: Details | undefined,
  fact: ModFact | undefined,
): Figures | undefined {
  const page = details?.page
  if (page) {
    const file = (details.files ?? []).find((f) => f.fileId === entry?.source.fileId)
    return {
      latest: page.version,
      released: file?.uploaded ?? '',
      updated: page.updated,
      rating: page.endorsements,
      downloads: page.downloads,
    }
  }
  if (!fact || (fact.latest === '' && fact.updated === '')) {
    return undefined
  }
  return {
    latest: fact.latest,
    released: fact.released,
    updated: fact.updated,
    rating: fact.rating,
    downloads: fact.downloads,
  }
}

/** A figure or fact column's text: the mod's own ids in its loader's terms (a SMAPI UniqueID, a BepInEx plugin GUID,
 * not Mortar's id for it), what its package ships, its plugins' versions, or one of its source's figures. */
export function figureText(
  row: ListRow,
  column: 'id' | 'kind' | 'pluginVersion' | 'latest' | 'released' | 'updated',
): string {
  switch (column) {
    case 'id':
      return (row.facts?.ids ?? []).join(', ')
    case 'kind':
      return row.facts?.kind ?? ''
    case 'pluginVersion':
      return (row.facts?.pluginVersions ?? []).join(', ')
    default:
      return row.figures?.[column] ?? ''
  }
}

/** The source's rating (Nexus endorsements, a Thunderstore rating) or downloads; a mod with endorsements of its own
 * and no figures keeps them. */
export function figureNumber(
  row: ListRow,
  column: 'endorsements' | 'downloads',
): number | undefined {
  if (column === 'downloads') {
    return row.figures?.downloads
  }
  if (row.figures) {
    return row.figures.rating
  }
  return row.mod.endorsements > 0 ? row.mod.endorsements : undefined
}
