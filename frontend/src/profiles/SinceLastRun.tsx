import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { History, List } from 'lucide-react'
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
import { HomePanel } from '../game/HomePanel.tsx'
import { changesView } from '../game/homeView.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { HistoryDialog } from './HistoryDialog.tsx'
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
  const { data: regressions } = useLoaded<StartupRegression[]>(
    () => {
      const [g = '', id = ''] = key.split('\0')
      return StartupReports(g, id).then((reports) => {
        const [latest, previous] = reports ?? []
        return latest ? startupRegressions(latest, previous) : []
      })
    },
    [key],
    [],
  )
  return regressions
}

function useSinceLastRun(game: string, profileId: string): string[] {
  const { t, i18n } = useLingui()
  const updated = useProfiles((s) => s.profiles.find((p) => p.id === profileId)?.updated ?? '')
  const [diff, setDiff] = useState<HistoryDiff | null>(null)
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
  return lines
}

// The latest changes as a short list; the full list and the history open on demand so a large change set never fills the page.
export function SinceLastRun({ game, profileId }: { game: string; profileId: string }) {
  const { t } = useLingui()
  const lines = useSinceLastRun(game, profileId)
  const [expanded, setExpanded] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  if (lines.length === 0) {
    return null
  }
  const { shown, canExpand } = changesView(lines, expanded)
  return (
    <HomePanel title={t`Changes since last run`} span={2}>
      <Box component="ul" sx={{ m: 0, p: 0, listStyle: 'none', width: '100%' }}>
        {shown.map((line) => (
          <Typography key={line} component="li" noWrap={true} title={line}>
            {line}
          </Typography>
        ))}
      </Box>
      <Box sx={{ display: 'flex', gap: 1 }}>
        {canExpand ? (
          <Button size="small" startIcon={<List size={14} />} onClick={() => setExpanded(true)}>
            {t`Show all`}
          </Button>
        ) : null}
        <Button size="small" startIcon={<History size={14} />} onClick={() => setHistoryOpen(true)}>
          {t`History…`}
        </Button>
      </Box>
      <HistoryDialog
        profileId={profileId}
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
      />
    </HomePanel>
  )
}
