import type { GameSettings } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import {
  SetListColumns,
  SetListSort,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { ListColumnId, ListColumnSort } from './listColumns.ts'

export function persistSort(sort: ListColumnSort) {
  useSettings.setState({ listSortColumn: sort.column, listSortDir: sort.dir })
  SetListSort(sort.column, sort.dir).catch(reportUnexpected)
}

// The choice belongs to the open game; a game without one follows the global default.
export function persistColumns(ids: ListColumnId[]) {
  const game = useProfiles.getState().game?.id
  if (!game) {
    return
  }
  useSettings.setState((s) => ({
    games: { ...s.games, [game]: { ...s.games?.[game], listColumns: ids } as GameSettings },
  }))
  SetListColumns(game, ids).catch(reportUnexpected)
}
