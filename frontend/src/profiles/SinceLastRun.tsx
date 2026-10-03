import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import { Runs } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import type { HistoryDiff } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { ChangesSince } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { diffLines } from './historyDiff.ts'

export function SinceLastRun({ game, profileId }: { game: string; profileId: string }) {
  const { t } = useLingui()
  const [diff, setDiff] = useState<HistoryDiff | null>(null)
  useEffect(() => {
    let live = true
    Runs(game, profileId)
      .then(async (runs) => {
        const since = runs?.[0]?.started ?? ''
        const next = await ChangesSince(game, profileId, since)
        if (live) {
          setDiff(next)
        }
      })
      .catch(() => {
        if (live) {
          setDiff(null)
        }
      })
    return () => {
      live = false
    }
  }, [game, profileId])
  const lines = diff ? diffLines(diff) : []
  if (lines.length === 0) {
    return null
  }
  return (
    <Box sx={{ px: 2, pt: 1.5, pb: 0 }}>
      <Box sx={{ p: 1.5, bgcolor: 'var(--mortar-paper-78)', borderRadius: '6px' }}>
        <Typography sx={{ fontWeight: 600, fontSize: 13 }}>{t`Since last run`}</Typography>
        <Box component="ul" sx={{ m: 0, mt: 0.75, pl: 2, fontSize: 13 }}>
          {lines.map((line) => (
            <li key={line}>{line}</li>
          ))}
        </Box>
      </Box>
    </Box>
  )
}
