import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Collapse, Typography } from '@mui/material'
import { ChevronDown, History } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  Runs,
  StartupReports,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import type { HistoryDiff } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ChangesSince } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  formatDuration,
  type StartupRegression,
  startupRegressions,
} from '../console/startupView.ts'
import { diffLines } from './historyDiff.ts'
import { useProfiles } from './store.ts'

const changesSinceCache = new Map<string, HistoryDiff | null>()

// Keyed by the profile's updated time, which changes after each run, when a new startup report may exist.
function useStartupRegressions(
  game: string,
  profileId: string,
  updated: string,
): StartupRegression[] {
  const key = `${game}\0${profileId}\0${updated}`
  const [regressions, setRegressions] = useState<StartupRegression[]>([])
  useEffect(() => {
    const [g = '', id = ''] = key.split('\0')
    let live = true
    StartupReports(g, id)
      .then((reports) => {
        const [latest, previous] = reports ?? []
        if (live) {
          setRegressions(latest ? startupRegressions(latest, previous) : [])
        }
      })
      .catch(() => {
        if (live) {
          setRegressions([])
        }
      })
    return () => {
      live = false
    }
  }, [key])
  return regressions
}

export function SinceLastRun({ game, profileId }: { game: string; profileId: string }) {
  const { t, i18n } = useLingui()
  const updated = useProfiles((s) => s.profiles.find((p) => p.id === profileId)?.updated ?? '')
  const [diff, setDiff] = useState<HistoryDiff | null>(null)
  const [open, setOpen] = useState(false)
  useEffect(() => {
    const key = `${game}\0${profileId}\0${updated}`
    if (changesSinceCache.has(key)) {
      setDiff(changesSinceCache.get(key) ?? null)
      return
    }
    let live = true
    Runs(game, profileId)
      .then(async (runs) => {
        const since = runs?.[0]?.started ?? ''
        const next = await ChangesSince(game, profileId, since)
        changesSinceCache.set(key, next)
        if (live) {
          setDiff(next)
        }
      })
      .catch(() => {
        changesSinceCache.set(key, null)
        if (live) {
          setDiff(null)
        }
      })
    return () => {
      live = false
    }
  }, [game, profileId, updated])
  const regressions = useStartupRegressions(game, profileId, updated)
  const lines = [
    ...regressions.map(
      (r) => t`Startup: ${r.name} ${r.version} added ${formatDuration(r.addedMs, i18n.locale)}`,
    ),
    ...(diff ? diffLines(diff) : []),
  ]
  if (lines.length === 0) {
    return null
  }
  // One row like the Problems bar; the full list opens on demand so a large change set never hides the page.
  return (
    <Box sx={{ mx: 2, mt: 1.25, flexShrink: 0 }}>
      <ButtonBase
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        sx={{
          width: '100%',
          display: 'flex',
          alignItems: 'center',
          gap: 1.25,
          height: 38,
          pl: 1.5,
          pr: 1,
          textAlign: 'left',
          bgcolor: (theme) =>
            theme.palette.mode === 'light'
              ? theme.palette.background.paper
              : 'var(--mortar-hairline-faint)',
          border: '1px solid var(--mortar-hairline-16)',
          borderRadius: '6px',
        }}
      >
        <Box component="span" sx={{ display: 'flex', flexShrink: 0, color: 'text.secondary' }}>
          <History size={16} aria-hidden={true} />
        </Box>
        <Typography component="span" sx={{ flexShrink: 0, fontSize: 14, fontWeight: 600 }}>
          {plural(lines.length, {
            one: 'Since last run: # change',
            other: 'Since last run: # changes',
          })}
        </Typography>
        <Typography
          component="span"
          noWrap={true}
          sx={{ flex: 1, minWidth: 0, fontSize: 14, color: 'text.secondary' }}
        >
          {lines[0]}
        </Typography>
        <Box
          component="span"
          sx={{
            display: 'flex',
            flexShrink: 0,
            transition: 'transform 150ms',
            transform: open ? 'rotate(180deg)' : 'none',
          }}
        >
          <ChevronDown size={16} aria-hidden={true} />
        </Box>
      </ButtonBase>
      <Collapse in={open} unmountOnExit={true}>
        <Box
          component="ul"
          aria-label={t`Changes since last run`}
          sx={{
            m: 0,
            mt: 0.5,
            py: 1,
            pl: 4,
            pr: 1.5,
            fontSize: 13,
            maxHeight: 240,
            overflowY: 'auto',
          }}
        >
          {lines.map((line) => (
            <li key={line}>{line}</li>
          ))}
        </Box>
      </Collapse>
    </Box>
  )
}
