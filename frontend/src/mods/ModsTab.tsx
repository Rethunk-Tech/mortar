import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Skeleton, Typography } from '@mui/material'
import { SearchX } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { useLaunch } from '../launch/store.ts'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { dialogOpen } from '../settings/shortcuts.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useCustomCategories } from './customCategories.ts'
import { DuplicateDialog } from './DuplicateDialog.tsx'
import { useDetail } from './detail.ts'
import { EndorsePrompt } from './EndorsePrompt.tsx'
import { LockedNote } from './LockedNote.tsx'
import { useLastRun } from './lastRun.ts'
import { useEntrySizes } from './listRows.ts'
import { entryOf, modId, modStatusProblem, updateFor } from './lookup.ts'
import { Cards } from './ModCards.tsx'
import { ModDetail } from './ModDetail.tsx'
import { ModList } from './ModList.tsx'
import { ModContextMenu } from './ModMenu.tsx'
import { useContextMenu } from './menu.ts'
import { ProblemBar } from './ProblemBar.tsx'
import { RemoveDialog } from './parts.tsx'
import { addedWithin, WEEK_MS } from './recent.ts'
import { SelectionBar } from './SelectionBar.tsx'
import { ModSidebar } from './Sidebar.tsx'
import { useSelection } from './selection.ts'
import { useMods, type View } from './store.ts'
import { EmptyMods, type ModFilter, Toolbar } from './Toolbar.tsx'
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
      if (e.key === 'Escape') {
        useSelection.getState().clear()
        return
      }
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'a') {
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
  query,
  onClear,
}: {
  profile: Profile
  shown: Mod[]
  view: View
  query: string
  onClear: () => void
}) {
  const { t } = useLingui()
  const loaded = useMods((s) => s.loaded)
  const loadError = useMods((s) => s.loadError)
  const load = useMods((s) => s.load)
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
  if (shown.length === 0 && query) {
    return (
      <EmptyState
        icon={<SearchX />}
        title={t`No mods match your search`}
        action={<Button onClick={onClear}>{t`Clear search`}</Button>}
      >
        {t`Try a different search.`}
      </EmptyState>
    )
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: 'minmax(0,1fr) auto' }}>
      <Box sx={{ minHeight: 0, display: 'flex', flexDirection: 'column' }}>
        {view === 'list' ? (
          <ModList profile={profile} mods={shown} />
        ) : (
          <Cards shown={shown} profile={profile} />
        )}
        <ModsFooter mods={shown} />
      </Box>
      <ModSidebar profile={profile} />
    </Box>
  )
}

// How many mods are shown and how much disk their entries take, once sizes have been measured.
function ModsFooter({ mods }: { mods: Mod[] }) {
  const { t } = useLingui()
  const sizes = useEntrySizes()
  const keys = new Set(mods.map((m) => m.key))
  let total = 0
  for (const key of keys) {
    total += sizes[key] ?? 0
  }
  const count = plural(mods.length, { one: '# mod', other: '# mods' })
  const label = total > 0 ? t`${count} · ${formatBytes(total)}` : count
  return (
    <Typography sx={{ px: 1.5, py: 0.75, fontSize: 12, color: 'text.secondary', flexShrink: 0 }}>
      {label}
    </Typography>
  )
}

// Placeholder rows while a profile's mods load for the first time.
const LOADING_ROW_COUNT = 8
const LOADING_ROWS = Array.from({ length: LOADING_ROW_COUNT }, (_, i) => i)
const ROW_HEIGHT = 44

export function ModsTab({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const view = useMods((s) => s.view)
  const load = useMods((s) => s.load)
  const query = useMods((s) => s.queries[profile.id] ?? '')
  const setQuery = useMods((s) => s.setQuery)
  const problems = useMods((s) => s.problems)
  const updates = useUpdates((s) => s.updates)
  const [filter, setFilter] = useState<ModFilter>('all')
  const loadKey = `${profile.id}:${profile.updated}`
  const gameId = useProfiles((s) => s.game?.id)
  const launchState = useLaunch((s) => s.status?.state)
  const loading = useRef(false)
  useEffect(() => {
    if (!gameId) {
      return
    }
    useCustomCategories.getState().load(gameId).catch(reportUnexpected)
  }, [gameId])

  useEffect(() => {
    if (!gameId || launchState === State.Launching || launchState === State.Running) {
      return
    }
    useLastRun
      .getState()
      .load(gameId, profile.id)
      .then(() => undefined)
  }, [gameId, profile.id, launchState])
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
      useUpdates.setState({ updates: null, reviewing: false })
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
    return <EmptyMods profileId={profile.id} />
  }

  const q = query.trim().toLowerCase()
  const shown = mods.filter((m) => {
    const entry = entryOf(profile, m.key)
    const searchable = [
      m.name,
      m.author,
      m.uniqueId,
      ...(entry?.tags ?? []),
      entry?.note ?? '',
      entry?.categoryOverride ?? '',
      JSON.stringify(entry?.source ?? ''),
    ]
    if (q && !searchable.some((value) => value.toLowerCase().includes(q))) {
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
        {t`Drop archives anywhere on the window, or Browse Nexus to find mods.`}
      </TipBanner>
      <ProblemBar />
      <UpdateBar />
      <Toolbar
        query={query}
        onQuery={(value) => setQuery(profile.id, value)}
        total={mods.length}
        filter={filter}
        onFilter={setFilter}
      />
      <LockedNote />
      <SelectionKeys shown={shown} />
      <SelectionBar profileId={profile.id} mods={shown} />
      <ModsBody
        profile={profile}
        shown={shown}
        view={view}
        query={q}
        onClear={() => setQuery(profile.id, '')}
      />
      <ModDetail profile={profile} />
      <UpdateReview profile={profile} />
      <ModContextMenu />
      <RemoveDialog />
      <DuplicateDialog profileName={profile.name} />
      <EndorsePrompt profile={profile} />
    </Box>
  )
}
