import { useDebouncedLoad } from '@rethunk/react-load-hooks'
import type {
  AssetMapPage,
  AssetTarget,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/models.ts'
import { AssetMap } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

const filterDelayMs = 200

// The asset map's pages for the filter, read a moment after typing stops; Show more appends the next page.
export function useAssetPages(open: boolean, filter: string, shared: boolean) {
  const game = useProfiles((s) => s.game)
  const openId = useProfiles((s) => s.openId)
  const at = (offset: number) => AssetMap(game?.id ?? '', openId, filter.trim(), shared, offset)
  const { data: page, run } = useDebouncedLoad<AssetMapPage | null>(
    open && game && openId ? () => at(0) : null,
    [open, game, openId, filter, shared],
    { initial: null, delayMs: filterDelayMs, onError: reportUnexpected },
  )
  const fetchFrom = (offset: number, before: AssetTarget[]) => {
    if (game && openId) {
      run(() =>
        at(offset).then((next) => ({ ...next, targets: [...before, ...(next.targets ?? [])] })),
      )
    }
  }
  const targets = page?.targets ?? []
  return {
    page,
    more: () => fetchFrom(targets.length, targets),
    reload: () => fetchFrom(0, []),
  }
}
