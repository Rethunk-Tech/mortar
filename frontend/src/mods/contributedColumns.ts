import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import type { ListColumnId } from './listColumns.ts'

interface Offer {
  deploy: string
  startup: boolean
  sources: ReadonlySet<string>
}

// Columns a deploy method, loader or source contributes appear only where the open profile has them; every other
// column is core.
const CONTRIBUTED: Partial<Record<ListColumnId, (o: Offer) => boolean>> = {
  order: (o) => o.deploy === 'profile',
  startup: (o) => o.startup,
  endorsements: (o) => o.sources.has('nexus'),
  downloads: (o) => o.sources.has('nexus'),
  updated: (o) => o.sources.has('nexus'),
}

export function useColumnAvailable(): (id: ListColumnId) => boolean {
  const game = useProfiles((s) => s.game)
  const profile = useProfiles(openProfileOf)
  const offer: Offer = {
    deploy: game?.deploy ?? '',
    startup: (game?.loaders ?? []).some((l) => l.startup),
    sources: new Set((profile?.entries ?? []).map((e) => e.source.kind)),
  }
  return (id) => CONTRIBUTED[id]?.(offer) ?? true
}

// The open game's own column choice, else the global default.
export function useSavedColumns(): readonly string[] | null | undefined {
  const game = useProfiles((s) => s.game?.id ?? '')
  return useSettings((s) => s.games?.[game]?.listColumns ?? s.listColumns)
}
