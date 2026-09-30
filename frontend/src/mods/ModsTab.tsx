import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Card, CircularProgress, Typography } from '@mui/material'
import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { userModCount } from '../profiles/count.ts'
import { openImport } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { DuplicateDialog } from './DuplicateDialog.tsx'
import { useDetail } from './detail.ts'
import { LockedNote } from './LockedNote.tsx'
import { modId } from './lookup.ts'
import { ModDetail } from './ModDetail.tsx'
import { ModList } from './ModList.tsx'
import { ModContextMenu, ModMenu } from './ModMenu.tsx'
import { contextMenuProps, useContextMenu } from './menu.ts'
import { ProblemBar } from './ProblemBar.tsx'
import { LetterTile, ProblemBadge, RemoveDialog, UpdateBadge } from './parts.tsx'
import { ModSidebar } from './Sidebar.tsx'
import { useMods, type View } from './store.ts'
import { AddArchive, BrowseNexus, Toolbar } from './Toolbar.tsx'
import { UpdateBar, UpdateReview } from './UpdateReview.tsx'
import { useUpdates } from './updates.ts'

const OFF_OPACITY = 0.6

function ModCard({ mod: m }: { mod: Mod }) {
  const { t } = useLingui()
  const openDetail = useDetail((s) => s.show)
  const selectedId = useDetail((s) => s.detailId)
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
        outline: modId(m) === selectedId ? '1px solid' : 'none',
        outlineColor: 'primary.main',
        [compact]: { height: 50, '& .tile': { width: 38, height: 38, fontSize: 19 } },
      }}
    >
      <ButtonBase
        aria-label={t`Details of ${m.name}`}
        onClick={() => openDetail(m)}
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
        <LetterTile mod={m} />
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
      <UpdateBadge mod={m} />
      <ProblemBadge mod={m} />
      <ModMenu mod={m} />
    </Card>
  )
}

function Cards({ shown }: { shown: Mod[] }) {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
        gap: '6px',
        px: 2,
        pt: '4px',
        pb: 1.75,
        overflowY: 'auto',
        alignContent: 'start',
      }}
    >
      {shown.map((m) => (
        <ModCard key={modId(m)} mod={m} />
      ))}
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
      {view === 'list' ? <ModList profile={profile} mods={shown} /> : <Cards shown={shown} />}
      <ModSidebar profile={profile} />
    </Box>
  )
}

export function ModsTab({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const view = useMods((s) => s.view)
  const load = useMods((s) => s.load)
  const [query, setQuery] = useState('')
  useEffect(() => {
    useMods.setState({
      mods: [],
      loaded: false,
      loadError: '',
      problems: null,
      resolving: null,
      removing: null,
    })
    useUpdates.setState({ updates: null, reviewing: false })
    useDetail.getState().show(null)
    useContextMenu.getState().close()
    load().catch(reportUnexpected)
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
      <ProblemBar />
      <UpdateBar />
      <Toolbar query={query} onQuery={setQuery} total={mods.length} />
      <LockedNote />
      <ModsBody profile={profile} shown={shown} view={view} query={q} />
      <ModDetail profile={profile} />
      <UpdateReview profile={profile} />
      <ModContextMenu />
      <RemoveDialog />
      <DuplicateDialog profileName={profile.name} />
    </Box>
  )
}
