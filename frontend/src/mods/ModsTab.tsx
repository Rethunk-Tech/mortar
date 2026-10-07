import { useLingui } from '@lingui/react/macro'
import { Box, Button, Skeleton, Typography } from '@mui/material'
import { SearchX } from 'lucide-react'
import { useEffect, useMemo, useRef } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useGameBusy } from '../launch/store.ts'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { boundShortcut, dialogOpen } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useCustomCategories } from './customCategories.ts'
import { DuplicateDialog } from './DuplicateDialog.tsx'
import { useDetail } from './detail.ts'
import { EndorsePrompt } from './EndorsePrompt.tsx'
import { customCategoryById, profileTags } from './group.ts'
import { LockedNote } from './LockedNote.tsx'
import { useLastRun } from './lastRun.ts'
import { entryOf, modId, modStatusProblem, updateFor } from './lookup.ts'
import { Cards } from './ModCards.tsx'
import { ModList } from './ModList.tsx'
import { ModContextMenu } from './ModMenu.tsx'
import { ModsHeaderActions } from './ModsHeaderActions.tsx'
import { useContextMenu } from './menu.ts'
import { hasAllTags, matchesQuery, searchFields } from './modSearch.ts'
import { NewFoldersCallout } from './NewFoldersCallout.tsx'
import { useNexusDetails } from './nexusDetails.ts'
import { OldFilesCallouts } from './OldFilesCallouts.tsx'
import { RemoveDialog } from './parts.tsx'
import { addedWithin, WEEK_MS } from './recent.ts'
import { SelectionBar } from './SelectionBar.tsx'
import { ModSidebar } from './Sidebar.tsx'
import { useSelection } from './selection.ts'
import { useMods } from './store.ts'
import { storedFilter, storedTags, type View } from './storeView.ts'
import { EmptyMods, Toolbar } from './Toolbar.tsx'
import { TrackedNotInProfile } from './TrackedNotInProfile.tsx'
import { ConfigPane } from './typedConfig/ConfigPane.tsx'
import { useTypedConfig } from './typedConfig/store.ts'
import { UpdateBar, UpdateReview } from './UpdateReview.tsx'
import { useUpdates } from './updates.ts'

function SelectionKeys({ shown }: { shown: Mod[] }) {
  useEffect(() => {
    useSelection.getState().prune(shown.map((m) => modId(m)))
  }, [shown])
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.defaultPrevented) {
        return
      }
      const el = e.target
      if (el instanceof HTMLElement && el.closest('input, textarea, [contenteditable="true"]')) {
        return
      }
      if (dialogOpen()) {
        return
      }
      const id = boundShortcut(e, useSettings.getState().shortcuts)
      // One press undoes one thing: a selection first, then the open details.
      if (id === 'dismiss') {
        const selection = useSelection.getState()
        if (selection.ids.length > 0) {
          selection.clear()
        } else {
          useDetail.getState().show(null)
        }
        return
      }
      if (id === 'select-all-mods') {
        e.preventDefault()
        useSelection.getState().selectAll(shown.map((m) => modId(m)))
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [shown])
  return null
}

function ModsBody({
  profile,
  shown,
  view,
  filtering,
  onClear,
}: {
  profile: Profile
  shown: Mod[]
  view: View
  filtering: boolean
  onClear: () => void
}) {
  const { t } = useLingui()
  const loaded = useMods((s) => s.loaded)
  const loadError = useMods((s) => s.loadError)
  const load = useMods((s) => s.load)
  const editingConfig = useTypedConfig((s) => s.mod !== null)
  if (editingConfig) {
    return <ConfigPane />
  }
  if (!loaded) {
    return loadError ? (
      <Box sx={{ px: 2, display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <Typography
          sx={{ color: 'error.main' }}
        >{t`Could not read the mods: ${loadError}`}</Typography>
        <Button variant="outlined" onClick={() => load().catch(reportUnexpected)}>
          {t`Retry`}
        </Button>
      </Box>
    ) : (
      <Box
        role="status"
        aria-label={t`Loading mods`}
        sx={{ px: 2, display: 'flex', flexDirection: 'column', gap: 1 }}
      >
        {LOADING_ROWS.map((n) => (
          <Skeleton
            key={n}
            variant="rounded"
            height={ROW_HEIGHT}
            sx={{ bgcolor: 'var(--mortar-hairline-faint)' }}
          />
        ))}
      </Box>
    )
  }
  if (shown.length === 0 && filtering) {
    return (
      <EmptyState
        icon={<SearchX />}
        title={t`No mods match`}
        action={<Button onClick={onClear}>{t`Show all mods`}</Button>}
      >
        {t`Try a different search or filter.`}
      </EmptyState>
    )
  }
  return (
    <Box
      onKeyDown={(e) => {
        // Escape belongs to a field or dialog first; otherwise it closes the details panel.
        const typing =
          e.target instanceof Element && e.target.closest('input, textarea, [role="dialog"]')
        if (e.key === 'Escape' && !typing && useDetail.getState().detailId !== '') {
          useDetail.getState().show(null)
        }
      }}
      sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: 'minmax(0,1fr) auto' }}
    >
      <Box sx={{ minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        {view === 'list' ? (
          <ModList profile={profile} mods={shown} />
        ) : (
          <Cards shown={shown} profile={profile} />
        )}
      </Box>
      <ModSidebar profile={profile} />
    </Box>
  )
}

// Placeholder rows while a profile's mods load for the first time.
const LOADING_ROW_COUNT = 8
const LOADING_ROWS = Array.from({ length: LOADING_ROW_COUNT }, (_, i) => i)
const ROW_HEIGHT = 44

function useSelectedTags(profileId: string): string[] {
  const saved = useMemo(() => storedTags(profileId), [profileId])
  return useMods((s) => s.tagFilters[profileId]) ?? saved
}

export function ModsTab({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const view = useMods((s) => s.view)
  const load = useMods((s) => s.load)
  const query = useMods((s) => s.queries[profile.id] ?? '')
  const setQuery = useMods((s) => s.setQuery)
  const problems = useMods((s) => s.problems)
  const updates = useUpdates((s) => s.updates)
  const filter = useMods((s) => s.filters[profile.id]) ?? storedFilter(profile.id)
  const setFilter = useMods((s) => s.setFilter)
  const selectedTags = useSelectedTags(profile.id)
  const setTagFilter = useMods((s) => s.setTagFilter)
  const nexusById = useNexusDetails((s) => s.byId)
  const customCategories = useCustomCategories((s) => s.categories)
  const loadKey = `${profile.id}:${profile.updated}`
  const gameId = useProfiles((s) => s.game?.id)
  const gameBusy = useGameBusy(gameId)
  const loading = useRef(false)
  useEffect(() => {
    if (!gameId) {
      return
    }
    useCustomCategories.getState().load(gameId).catch(reportUnexpected)
  }, [gameId])

  useEffect(() => {
    if (!gameId || gameBusy) {
      return
    }
    useLastRun
      .getState()
      .load(gameId, profile.id)
      .then(() => undefined)
  }, [gameId, profile.id, gameBusy])
  useEffect(() => {
    if (loadKey === '') {
      return
    }
    const pending = useDetail.getState().pendingId
    if (useMods.getState().modsFor !== profile.id) {
      useMods.setState({
        mods: [],
        loaded: false,
        loadError: '',
        problems: null,
        resolving: null,
        removing: [],
      })
      useUpdates.setState({ updates: null })
    }
    if (!pending) {
      useDetail.getState().show(null)
    }
    useContextMenu.getState().close()
    useSelection.getState().clear()
    if (loading.current) {
      return
    }
    loading.current = true
    load()
      .then(() => {
        const id = useDetail.getState().takePending()
        if (id) {
          useDetail.getState().show(useMods.getState().mods.find((m) => modId(m) === id) ?? null)
        }
      })
      .catch(reportUnexpected)
      .finally(() => {
        loading.current = false
      })
  }, [load, loadKey, profile.id])

  if (userModCount(profile) === 0) {
    return (
      <>
        <TrackedNotInProfile profile={profile} />
        <OldFilesCallouts profile={profile} />
        <NewFoldersCallout profile={profile} />
        <EmptyMods profileId={profile.id} />
      </>
    )
  }

  const q = query.trim().toLowerCase()
  const customById = customCategoryById(customCategories)
  const shown = mods.filter((m) => {
    const entry = entryOf(profile, m.key)
    const searchable = searchFields(m, entry, nexusById, customById)
    if (!(matchesQuery(q, searchable) && hasAllTags(entry?.tags, selectedTags))) {
      return false
    }
    return (
      filter === 'all' ||
      (filter === 'disabled' && !m.enabled) ||
      (filter === 'update' && Boolean(updateFor(updates, m, profile))) ||
      (filter === 'problem' && modStatusProblem(problems, m)) ||
      (filter === 'pinned' && Boolean(entry?.pinned)) ||
      (filter === 'local' && entry?.source.kind === 'local') ||
      (filter === 'recent' && addedWithin(entry?.added, WEEK_MS, Date.now()))
    )
  })
  return (
    <Box
      sx={{ position: 'relative', flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
    >
      <TipBanner tip="mods">
        {t`Drop archives anywhere on the window, or use Browse to find mods.`}
      </TipBanner>
      <ModsHeaderActions />
      <UpdateBar />
      <OldFilesCallouts profile={profile} />
      <NewFoldersCallout profile={profile} />
      <Toolbar
        query={query}
        onQuery={(value) => setQuery(profile.id, value)}
        total={mods.length}
        filter={filter}
        onFilter={(value) => setFilter(profile.id, value)}
        tags={profileTags(profile.entries)}
        selectedTags={selectedTags}
        onTags={(tags) => setTagFilter(profile.id, tags)}
      />
      <TrackedNotInProfile profile={profile} />
      <LockedNote />
      <SelectionKeys shown={shown} />
      <SelectionBar profileId={profile.id} mods={shown} />
      <ModsBody
        profile={profile}
        shown={shown}
        view={view}
        filtering={q !== '' || filter !== 'all' || selectedTags.length > 0}
        onClear={() => {
          setQuery(profile.id, '')
          setFilter(profile.id, 'all')
          setTagFilter(profile.id, [])
        }}
      />
      <UpdateReview profile={profile} />
      <ModContextMenu />
      <RemoveDialog />
      <DuplicateDialog profileName={profile.name} />
      <EndorsePrompt profile={profile} />
    </Box>
  )
}
