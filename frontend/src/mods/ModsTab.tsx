import { useLingui } from '@lingui/react/macro'
import { Box, Card, Typography } from '@mui/material'
import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { ModList } from './ModList.tsx'
import { LetterTile, ModMenu, ModSwitch, RemoveDialog } from './parts.tsx'
import { useMods, type View } from './store.ts'
import { AddArchive, BrowseNexus, Toolbar } from './Toolbar.tsx'

const OFF_OPACITY = 0.6

function ModsBody({ profile, shown, view }: { profile: Profile; shown: Mod[]; view: View }) {
  const { t } = useLingui()
  if (shown.length === 0) {
    return (
      <Typography
        sx={{ px: 2, color: 'text.secondary' }}
      >{t`No mods match your search.`}</Typography>
    )
  }
  if (view === 'list') {
    return <ModList profile={profile} mods={shown} />
  }
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
        gap: '6px',
        px: 2,
        pb: 1.75,
        overflowY: 'auto',
        alignContent: 'start',
      }}
    >
      {shown.map((m) => (
        <Card
          key={m.uniqueId}
          sx={{
            height: 64,
            pl: 1,
            pr: 0.75,
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            minWidth: 0,
            borderRadius: '6px',
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
          <ModSwitch mod={m} />
          <ModMenu mod={m} />
        </Card>
      ))}
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
    useMods.setState({ mods: [], loaded: false })
    load().catch(reportUnexpected)
  }, [load])

  if ((profile.entries ?? []).length === 0) {
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
      </Box>
    )
  }

  const q = query.trim().toLowerCase()
  const shown = mods.filter(
    (m) => !q || m.name.toLowerCase().includes(q) || m.author.toLowerCase().includes(q),
  )
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Toolbar query={query} onQuery={setQuery} total={mods.length} />
      <ModsBody profile={profile} shown={shown} view={view} />
      <RemoveDialog />
    </Box>
  )
}
