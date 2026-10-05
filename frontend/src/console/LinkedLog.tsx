import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import type { Entry } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ModsDir,
  OpenConsolePath,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useTab } from '../game/tab.ts'
import { useDetail } from '../mods/detail.ts'
import { useMods } from '../mods/store.ts'
import { useUpdates } from '../mods/updates.ts'
import { useProfiles } from '../profiles/store.ts'
import { MONO } from '../theme/theme.ts'
import { toastError } from '../toasts/report.ts'
import type { InstalledMod } from './consoleLinks.ts'
import { smapiUpdateNotes } from './smapiUpdateNotes.ts'
import { VirtualLog } from './VirtualLog.tsx'

function installedOf(
  entries: { mods?: { name: string; uniqueId: string }[] | null }[] | null | undefined,
): InstalledMod[] {
  const out: InstalledMod[] = []
  for (const entry of entries ?? []) {
    for (const m of entry.mods ?? []) {
      out.push({ name: m.name, uniqueID: m.uniqueId })
    }
  }
  return out
}

function openMod(uniqueID: string) {
  useTab.getState().setTab('mods')
  const shown = (mod: Mod | undefined) => {
    if (!mod) {
      return
    }
    useDetail.getState().show(mod)
    useDetail.getState().setOpen(true)
  }
  const found = useMods.getState().mods.find((m) => m.uniqueId === uniqueID)
  if (found) {
    shown(found)
    return
  }
  useMods
    .getState()
    .load()
    .then(() => shown(useMods.getState().mods.find((m) => m.uniqueId === uniqueID)))
}

export function LinkedLog({
  game,
  rows,
  timestamps,
  follow,
  jump,
  empty,
  onUnfollow,
}: {
  game: string
  rows: Entry[]
  timestamps: boolean
  follow: boolean
  jump: { index: number; n: number } | null
  empty: string | null
  onUnfollow: () => void
}) {
  const { t } = useLingui()
  const profile = useProfiles((s) => s.openId)
  const gameDir = useProfiles((s) => s.game?.installDir ?? '')
  const profileEntries = useProfiles((s) => s.profiles.find((p) => p.id === s.openId)?.entries)
  const installed = useMemo(() => installedOf(profileEntries), [profileEntries])
  const updates = useUpdates((s) => s.updates)
  const notes = useMemo(() => smapiUpdateNotes(rows, updates), [rows, updates])
  const [modsDir, setModsDir] = useState('')
  useEffect(() => {
    if (profile === '') {
      setModsDir('')
      return
    }
    let live = true
    ModsDir(game, profile)
      .then((dir) => {
        if (live) {
          setModsDir(dir ?? '')
        }
      })
      .catch(() => {
        if (live) {
          setModsDir('')
        }
      })
    return () => {
      live = false
    }
  }, [game, profile])
  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        mx: 2,
        mb: 1.5,
        bgcolor: (theme) =>
          theme.palette.mode === 'light'
            ? theme.palette.background.paper
            : 'var(--mortar-overlay-50)',
        borderRadius: '6px',
        fontFamily: MONO,
        fontSize: 13,
        lineHeight: '23px',
        overflow: 'hidden',
      }}
    >
      {empty ? (
        <Typography sx={{ p: 2, font: 'inherit', color: 'text.secondary' }}>{empty}</Typography>
      ) : (
        <VirtualLog
          rows={rows}
          notes={notes}
          timestamps={timestamps}
          follow={follow}
          onUnfollow={onUnfollow}
          jump={jump}
          mods={installed}
          roots={{ modsDir, gameDir }}
          onMod={openMod}
          onPath={(path) => {
            OpenConsolePath(game, profile, path).catch((e: unknown) => {
              toastError(t`Could not open the folder`, e)
            })
          }}
        />
      )}
    </Box>
  )
}
