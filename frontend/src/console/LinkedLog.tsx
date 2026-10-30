import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { type ReactNode, useCallback, useMemo } from 'react'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import {
  ModsDir,
  OpenConsolePath,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openMod } from '../mods/openMod.ts'
import { useUpdates } from '../mods/updates.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { reportError, reportUnexpected } from '../toasts/report.ts'
import type { InstalledMod } from './consoleLinks.ts'
import { smapiUpdateNotes } from './smapiUpdateNotes.ts'
import { VirtualLog } from './VirtualLog.tsx'

function installedOf(
  entries: { mods?: { name: string; id: string }[] | null }[] | null | undefined,
): InstalledMod[] {
  const out: InstalledMod[] = []
  for (const entry of entries ?? []) {
    for (const m of entry.mods ?? []) {
      out.push({ name: m.name, id: m.id })
    }
  }
  return out
}

export function LinkedLog({
  game,
  rows,
  timestamps,
  follow,
  jump,
  empty,
}: {
  game: string
  rows: Entry[]
  timestamps: boolean
  follow: boolean
  jump: { index: number; n: number } | null
  empty: ReactNode
}) {
  const { t } = useLingui()
  const profile = useProfiles((s) => s.openId)
  const gameDir = useProfiles((s) => s.game?.installDir ?? '')
  const profileEntries = useProfiles((s) => openProfileOf(s)?.entries)
  const installed = useMemo(() => installedOf(profileEntries), [profileEntries])
  const updates = useUpdates((s) => s.updates)
  const notes = useMemo(() => smapiUpdateNotes(rows, updates), [rows, updates])
  const { data: modsDir } = useLoaded(
    profile === '' ? null : () => ModsDir(game, profile).then((dir) => dir ?? ''),
    [game, profile],
    '',
  )
  // Kept between renders, so the memoised rows in view skip a render when lines arrive.
  const roots = useMemo(() => ({ modsDir, gameDir }), [modsDir, gameDir])
  const onMod = useCallback((id: string) => {
    openMod({ id }).catch(reportUnexpected)
  }, [])
  const onPath = useCallback(
    (path: string) => {
      OpenConsolePath(game, profile, path).catch(reportError(t`Could not open the folder`))
    },
    [game, profile, t],
  )
  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        mx: space.gutter,
        mb: 1.5,
        bgcolor: (theme) =>
          theme.palette.mode === 'light'
            ? theme.palette.background.paper
            : 'var(--mortar-overlay-50)',
        borderRadius: '6px',
        fontFamily: MONO,
        fontSize: 'base.fontSize',
        lineHeight: '23px',
        overflow: 'hidden',
      }}
    >
      {empty ? (
        <Typography component="div" sx={{ p: space.pad, font: 'inherit', color: 'text.secondary' }}>
          {empty}
        </Typography>
      ) : (
        <VirtualLog
          game={game}
          rows={rows}
          notes={notes}
          timestamps={timestamps}
          follow={follow}
          jump={jump}
          mods={installed}
          roots={roots}
          onMod={onMod}
          onPath={onPath}
        />
      )}
    </Box>
  )
}
