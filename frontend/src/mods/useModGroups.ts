import { useEffect, useMemo, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
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
import { toListRows, useEntrySizes, useStartupCosts } from './listRows.ts'
import { modId, modStatusProblem, nexusIdOf, updateFor } from './lookup.ts'
import { primeDetails, useNexusDetails } from './nexusDetails.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

// The list's and the cards' rows, grouped and sorted by the user's settings, rebuilt only when an input changes.
export function useModGroups(mods: Mod[], profile: Profile) {
  const groupBy = sanitizeListGroupBy(useSettings((s) => s.listGroupBy))
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
  const problems = useMods((s) => s.problems)
  const updates = useUpdates((s) => s.updates)
  useEffect(() => {
    setCollapsed(loadCollapsed(gameId))
  }, [gameId])
  useEffect(() => {
    primeDetails(mods.map((m) => nexusIdOf(profile, m)).filter((id) => id > 0)).catch(
      reportUnexpected,
    )
  }, [mods, profile])
  const groups = useMemo(() => {
    const names = installedNames(mods)
    return groupSorted(
      toListRows(mods, profile, { byId, customById, sizes, costs }),
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
  }, [mods, profile, byId, groupBy, problems, updates, customById, sort, sizes, costs])
  const orderedIds = useMemo(
    () => groups.flatMap((g) => g.items.map((r) => modId(r.mod))),
    [groups],
  )
  return { groupBy, sort, gameId, collapsed, setCollapsed, groups, orderedIds }
}
