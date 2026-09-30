import { useLingui } from '@lingui/react/macro'
import { Box, Card, Typography } from '@mui/material'
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

function ModsBody({ profile, shown, view }: { profile: Profile; shown: Mod[]; view: View }) {
  const { t } = useLingui()
  if (shown.length === 0) {
    return <Typography sx={{ color: 'text.secondary' }}>{t`No mods match your search.`}</Typography>
  }
  if (view === 'list') {
    return <ModList profile={profile} mods={shown} />
  }
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))',
        gap: 2,
        overflowY: 'auto',
        alignContent: 'start',
      }}
    >
      {shown.map((m) => (
        <Card
          key={m.uniqueId}
          sx={{ p: 1.5, display: 'flex', alignItems: 'center', gap: 1.5, minWidth: 0 }}
        >
          <LetterTile mod={m} />
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography noWrap={true} sx={{ fontWeight: 600 }}>
              {m.name}
            </Typography>
            <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
              {m.author}
            </Typography>
            <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
              {m.version}
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
        sx={{ p: 3, display: 'flex', flexDirection: 'column', alignItems: 'flex-start', gap: 1.5 }}
      >
        <Typography sx={{ fontSize: 20, fontWeight: 600 }}>{t`No mods yet`}</Typography>
        <Typography sx={{ color: 'text.secondary' }}>
          {t`Add mods from an archive you downloaded, or find them on Nexus.`}
        </Typography>
        <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
          <AddArchive variant="contained" />
          <BrowseNexus variant="outlined" />
        </Box>
        <Typography sx={{ color: 'text.secondary' }}>
          {t`You can also drop archives anywhere on the window.`}
        </Typography>
      </Box>
    )
  }

  const q = query.trim().toLowerCase()
  const shown = mods.filter(
    (m) => !q || m.name.toLowerCase().includes(q) || m.author.toLowerCase().includes(q),
  )
  return (
    <Box sx={{ p: 3, flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Toolbar query={query} onQuery={setQuery} />
      <ModsBody profile={profile} shown={shown} view={view} />
      <RemoveDialog />
    </Box>
  )
}
