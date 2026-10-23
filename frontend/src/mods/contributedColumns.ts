import { openProfileOf, useProfileLoader, useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import type { ListColumnId } from './listColumns.ts'
import { useModFacts } from './listFigures.ts'

interface Offer {
  deploy: string
  startup: boolean
  sources: ReadonlySet<string>
  /** The loader's term for its mods' ids; empty when they carry none. */
  idName: string
  /** Some mod in the profile is a BepInEx package whose files Mortar has read (plugin, patcher, library). */
  kinds: boolean
}

// The sources whose figures (latest, dates, rating, downloads) Mortar reads without a request per mod.
const FIGURE_SOURCES = ['nexus', 'thunderstore']
const figures = (o: Offer) => FIGURE_SOURCES.some((s) => o.sources.has(s))

// Columns a deploy method, loader or source contributes appear only where the open profile has them; every other
// column is core.
const CONTRIBUTED: Partial<Record<ListColumnId, (o: Offer) => boolean>> = {
  order: (o) => o.deploy === 'profile',
  startup: (o) => o.startup,
  id: (o) => o.idName !== '',
  kind: (o) => o.kinds,
  pluginVersion: (o) => o.kinds,
  latest: figures,
  released: figures,
  updated: figures,
  endorsements: figures,
  downloads: figures,
}

export function useColumnAvailable(): (id: ListColumnId) => boolean {
  const game = useProfiles((s) => s.game)
  const profile = useProfiles(openProfileOf)
  const kinds = useModFacts((s) => Object.values(s.byId).some((f) => (f?.kind ?? '') !== ''))
  const offer: Offer = {
    deploy: game?.deploy ?? '',
    startup: (game?.loaders ?? []).some((l) => l.startup),
    sources: new Set((profile?.entries ?? []).map((e) => e.source.kind)),
    idName: useModIdName(),
    kinds,
  }
  return (id) => CONTRIBUTED[id]?.(offer) ?? true
}

// The open game's own column choice, else the global default.
export function useSavedColumns(): readonly string[] | null | undefined {
  const game = useProfiles((s) => s.game?.id ?? '')
  return useSettings((s) => s.games?.[game]?.listColumns ?? s.listColumns)
}

/** The open game's loader's term for its mods' ids (UniqueID, GUID), empty when they carry none. */
export function useModIdName(): string {
  return useProfileLoader()?.modIdName ?? ''
}
