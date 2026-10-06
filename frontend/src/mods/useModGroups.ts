import { useEffect, useMemo, useRef, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfileLoader, useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useCustomCategories } from './customCategories.ts'
import {
  customCategoryById,
  groupSorted,
  installedNames,
  loadCollapsed,
  rowGroupKey,
  sanitizeListGroupBy,
} from './group.ts'
import { compareListRows, sanitizeListSort } from './listColumns.ts'
import { toListRows, useEntrySizes, usePackageOverrides, useStartupCosts } from './listRows.ts'
import { modId, modStatusProblem, nexusIdOf, updateFor } from './lookup.ts'
import { primeDetails, useNexusDetails } from './nexusDetails.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

// The list's and the cards' rows, grouped and sorted by the user's settings, rebuilt only when an input changes.
export function useModGroups(mods: Mod[], profile: Profile) {
  const groupBy = useListGroupBy()
  const listSortColumn = useSettings((s) => s.listSortColumn)
  const listSortDir = useSettings((s) => s.listSortDir)
  const sort = useMemo(
    () => sanitizeListSort(listSortColumn ?? '', listSortDir ?? ''),
    [listSortColumn, listSortDir],
  )
  const gameId = useProfiles((s) => s.game?.id) ?? ''
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>(() => loadCollapsed(gameId))
  const byId = useNexusDetails((s) => s.byId)
  const customCategories = useCustomCategories((s) => s.categories)
  const customById = useMemo(() => customCategoryById(customCategories), [customCategories])
  const sizes = useEntrySizes()
  const costs = useStartupCosts(gameId, profile.id)
  const wins = usePackageOverrides(gameId, profile, useProfiles((s) => s.game?.deploy) ?? '')
  const problems = useMods((s) => s.problems)
  const updates = useUpdates((s) => s.updates)
  useEffect(() => {
    setCollapsed(loadCollapsed(gameId))
  }, [gameId])
  const nexusIds = mods
    .map((m) => nexusIdOf(profile, m))
    .filter((id) => id > 0)
    .sort((a, b) => a - b)
    .join(',')
  useEffect(() => {
    primeDetails(nexusIds === '' ? [] : nexusIds.split(',').map(Number)).catch(reportUnexpected)
  }, [nexusIds])
  const groups = useMemo(() => {
    const names = installedNames(mods)
    return groupSorted(
      toListRows(mods, profile, { byId, customById, sizes, costs, wins }),
      groupBy,
      (row) =>
        rowGroupKey(groupBy, row, {
          hasProblem: modStatusProblem(problems, row.mod),
          hasUpdate: Boolean(updateFor(updates, row.mod, profile)),
          names,
          customById,
        }),
      (a, b) => compareListRows(a, b, sort),
    )
  }, [mods, profile, byId, groupBy, problems, updates, customById, sort, sizes, costs, wins])
  // Rows are memoised on this list's identity, so an unchanged order keeps the previous array.
  const lastIds = useRef<string[]>([])
  const orderedIds = useMemo(() => {
    const next = groups.flatMap((g) => g.items.map((r) => modId(r.mod)))
    const last = lastIds.current
    if (next.length === last.length && next.every((id, i) => id === last[i])) {
      return last
    }
    lastIds.current = next
    return next
  }, [groups])
  return { groupBy, sort, gameId, collapsed, setCollapsed, groups, orderedIds }
}

// The saved grouping, or Status when it is Framework and the profile's loader has no framework mods to group by.
export function useListGroupBy() {
  const saved = sanitizeListGroupBy(useSettings((s) => s.listGroupBy))
  const frameworks = useProfileLoader()?.frameworks ?? false
  return saved === 'framework' && !frameworks ? 'status' : saved
}
