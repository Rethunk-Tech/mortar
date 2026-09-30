import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Card, Chip, CircularProgress, Typography } from '@mui/material'
import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { useLaunch } from '../launch/store.ts'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { dialogOpen } from '../settings/shortcuts.ts'
import { useSettings } from '../settings/store.ts'
import { openImport } from '../share/store.ts'
import { TipBanner } from '../tips/TipBanner.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { DuplicateDialog } from './DuplicateDialog.tsx'
import { useDetail } from './detail.ts'
import { emptyGroupLabel, firstTag, groupSorted, sanitizeListGroupBy } from './group.ts'
import { LockedNote } from './LockedNote.tsx'
import { useLastRun } from './lastRun.ts'
import { compareListRows, sanitizeListSort } from './listColumns.ts'
import { toListRow } from './listRows.ts'
import { entryOf, modId, nexusIdOf } from './lookup.ts'
import { ModDetail } from './ModDetail.tsx'
import { ModList } from './ModList.tsx'
import { ModContextMenu, ModMenu } from './ModMenu.tsx'
import { ModsGroupHeader } from './ModsGroupHeader.tsx'
import { contextMenuProps, useContextMenu } from './menu.ts'
import { primeDetails, useNexusDetails, useNexusFresh } from './nexusDetails.ts'
import { ProblemBar } from './ProblemBar.tsx'
import {
  LastRunBadge,
  LetterTile,
  PinBadge,
  ProblemBadge,
  RemoveDialog,
  UpdateBadge,
} from './parts.tsx'
import { SelectionBar } from './SelectionBar.tsx'
import { ModSidebar } from './Sidebar.tsx'
import { useSelection } from './selection.ts'
import { useMods, type View } from './store.ts'
import { AddArchive, BrowseNexus, Toolbar } from './Toolbar.tsx'
import { UpdateBar, UpdateReview } from './UpdateReview.tsx'
import { useUpdates } from './updates.ts'

const OFF_OPACITY = 0.6

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

function ModCard({
  mod: m,
  orderedIds,
  profile,
}: {
  mod: Mod
  orderedIds: readonly string[]
  profile: Profile
}) {
  const { t } = useLingui()
  const openDetail = useDetail((s) => s.show)
  const selectedId = useDetail((s) => s.detailId)
  const selectedIds = useSelection((s) => s.ids)
  const id = modId(m)
  const marked = selectedIds.includes(id) || (selectedIds.length === 0 && id === selectedId)
  const fresh = useNexusFresh(nexusIdOf(profile, m))
  const tag = firstTag(entryOf(profile, m.key)?.tags)
  return (
    <Card
      {...contextMenuProps(m)}
      sx={{
        height: 64,
        pl: 1,
        pr: 0.75,
        display: 'flex',
        alignItems: 'center',
        gap: '10px',
        minWidth: 0,
        borderRadius: '6px',
        outline: marked ? '1px solid' : 'none',
        outlineColor: 'primary.main',
        [compact]: { height: 50, '& .tile': { width: 38, height: 38, fontSize: 19 } },
      }}
    >
      <ButtonBase
        aria-label={t`Details of ${m.name}`}
        onMouseDown={(e) => {
          if (e.shiftKey) {
            e.preventDefault()
          }
        }}
        onClick={(e) => {
          useSelection.getState().click(orderedIds, id, e)
          openDetail(m)
        }}
        sx={{
          '&.Mui-focusVisible': { outlineOffset: '-2px' },
          flex: 1,
          minWidth: 0,
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: '10px',
          justifyContent: 'flex-start',
          textAlign: 'left',
          fontFamily: 'inherit',
          color: 'inherit',
        }}
      >
        <LetterTile mod={m} fresh={fresh} />
        <Box
          sx={{
            flex: 1,
            minWidth: 0,
            pl: '10px',
            borderLeft: '1px solid rgba(255,255,255,0.12)',
            opacity: m.enabled ? 1 : OFF_OPACITY,
          }}
        >
          <Typography noWrap={true} sx={{ fontSize: 14, fontWeight: 600 }}>
            {m.name}
          </Typography>
          <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary' }}>
            {`${m.author} · ${m.version}`}
          </Typography>
        </Box>
      </ButtonBase>
      <PinBadge mod={m} />
      <UpdateBadge mod={m} />
      <ProblemBadge mod={m} />
      <LastRunBadge mod={m} />
      {tag ? <Chip size="small" label={tag} sx={{ maxWidth: 96 }} /> : null}
      <ModMenu mod={m} />
    </Card>
  )
}

function Cards({ shown, profile }: { shown: Mod[]; profile: Profile }) {
  const { t } = useLingui()
  const groupBy = sanitizeListGroupBy(useSettings((s) => s.listGroupBy))
  const listSortColumn = useSettings((s) => s.listSortColumn)
  const listSortDir = useSettings((s) => s.listSortDir)
  const sort = sanitizeListSort(listSortColumn ?? '', listSortDir ?? '')
  const byId = useNexusDetails((s) => s.byId)
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const tagHint = t`A mod with several tags appears under its first tag.`
  useEffect(() => {
    primeDetails(shown.map((m) => nexusIdOf(profile, m)).filter((id) => id > 0)).catch(
      reportUnexpected,
    )
  }, [shown, profile])
  const groups = groupSorted(
    shown.map((m) => toListRow(m, profile, byId)),
    groupBy,
    (row) => {
      if (groupBy === 'category') {
        return row.details?.category ?? ''
      }
      if (groupBy === 'source') {
        return row.source
      }
      if (groupBy === 'tag') {
        return firstTag(row.tags)
      }
      return ''
    },
    (a, b) => compareListRows(a, b, sort),
  )
  const orderedIds = groups.flatMap((g) => g.items.map((r) => modId(r.mod)))
  const emptyLabel = emptyGroupLabel(groupBy, {
    category: t`Uncategorised`,
    source: t`Unknown source`,
    tag: t`Untagged`,
  })
  return (
    <Box sx={{ minHeight: 0, overflowY: 'auto' }}>
      {groups.map((group) => {
        const open = collapsed[group.key] !== true
        return (
          <Box key={group.key || 'none'}>
            {groupBy === 'none' ? null : (
              <ModsGroupHeader
                label={group.key || emptyLabel}
                count={group.items.length}
                open={open}
                onToggle={() => setCollapsed((cur) => ({ ...cur, [group.key]: open }))}
                {...(groupBy === 'tag' ? { hint: tagHint } : {})}
              />
            )}
            {open ? (
              <Box
                sx={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
                  gap: '6px',
                  px: 2,
                  pt: '4px',
                  pb: 1.75,
                  alignContent: 'start',
                }}
              >
                {group.items.map((r) => (
                  <ModCard
                    key={modId(r.mod)}
                    mod={r.mod}
                    orderedIds={orderedIds}
                    profile={profile}
                  />
                ))}
              </Box>
            ) : null}
          </Box>
        )
      })}
    </Box>
  )
}

function ModsBody({
  profile,
  shown,
  view,
  query,
}: {
  profile: Profile
  shown: Mod[]
  view: View
  query: string
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
      <Box sx={{ px: 2 }}>
        <CircularProgress size={20} aria-label={t`Loading mods`} />
      </Box>
    )
  }
  if (shown.length === 0 && query) {
    return (
      <Typography
        sx={{ px: 2, color: 'text.secondary' }}
      >{t`No mods match your search.`}</Typography>
    )
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: 'minmax(0,1fr) auto' }}>
      {view === 'list' ? (
        <ModList profile={profile} mods={shown} />
      ) : (
        <Cards shown={shown} profile={profile} />
      )}
      <ModSidebar profile={profile} />
    </Box>
  )
}

export function ModsTab({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const view = useMods((s) => s.view)
  const load = useMods((s) => s.load)
  const gameId = useProfiles((s) => s.game?.id)
  const launchState = useLaunch((s) => s.status?.state)
  const [query, setQuery] = useState('')
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
    const pending = useDetail.getState().pendingId
    useMods.setState({
      mods: [],
      loaded: false,
      loadError: '',
      problems: null,
      resolving: null,
      removing: [],
    })
    useUpdates.setState({ updates: null, reviewing: false })
    if (!pending) {
      useDetail.getState().show(null)
    }
    useContextMenu.getState().close()
    useSelection.getState().clear()
    load()
      .then(() => {
        const id = useDetail.getState().takePending()
        if (!id) {
          return
        }
        const mod = useMods.getState().mods.find((m) => modId(m) === id)
        useDetail.getState().show(mod ?? null)
      })
      .catch(reportUnexpected)
  }, [load])

  if (userModCount(profile) === 0) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 2.25,
          px: 3,
          textAlign: 'center',
        }}
      >
        <Typography sx={{ fontSize: 26, fontWeight: 700 }}>{t`No mods yet`}</Typography>
        <TipBanner tip="mods">
          {t`Drop archives anywhere on the window, or Browse Nexus to find mods.`}
        </TipBanner>
        <Typography sx={{ maxWidth: 520, fontSize: 15, lineHeight: 1.5 }}>
          {t`Add mods from an archive you downloaded, or find them on Nexus.`}
        </Typography>
        <Box sx={{ display: 'flex', gap: 1.5 }}>
          <BrowseNexus variant="contained" size="large" />
          <AddArchive variant="outlined" size="large" />
        </Box>
        <Box
          sx={{
            mt: 1.5,
            px: 1.75,
            py: 1.25,
            display: 'flex',
            alignItems: 'center',
            gap: 1.25,
            fontSize: 14,
            border: '1px dashed rgba(255,255,255,0.25)',
            borderRadius: '8px',
          }}
        >
          <Download size={16} aria-hidden={true} />
          {t`You can also drop archives anywhere on the window.`}
        </Box>
        <Button
          variant="text"
          onClick={() => openImport({ profileId: profile.id })}
          sx={{ textDecoration: 'underline' }}
        >
          {t`Or import a shared profile`}
        </Button>
      </Box>
    )
  }

  const q = query.trim().toLowerCase()
  const shown = mods.filter(
    (m) => !q || m.name.toLowerCase().includes(q) || m.author.toLowerCase().includes(q),
  )
  return (
    <Box
      sx={{ position: 'relative', flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
    >
      <TipBanner tip="mods">
        {t`Drop archives anywhere on the window, or Browse Nexus to find mods.`}
      </TipBanner>
      <ProblemBar />
      <UpdateBar />
      <Toolbar query={query} onQuery={setQuery} total={mods.length} />
      <LockedNote />
      <SelectionKeys shown={shown} />
      <SelectionBar profileId={profile.id} mods={shown} />
      <ModsBody profile={profile} shown={shown} view={view} query={q} />
      <ModDetail profile={profile} />
      <UpdateReview profile={profile} />
      <ModContextMenu />
      <RemoveDialog />
      <DuplicateDialog profileName={profile.name} />
    </Box>
  )
}
