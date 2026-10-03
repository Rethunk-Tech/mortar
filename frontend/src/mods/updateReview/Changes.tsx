import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Changelog } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/nexus/models.ts'
import { ChangelogBetween } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import type { Update } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { Fold } from '../../shell/Fold.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { changelogNoteIsRisky } from '../changelogRange.ts'

export function Changes({ update }: { update: Update }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [logs, setLogs] = useState<Changelog[] | undefined>(undefined)
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    if (!(update.nexusId > 0 && game)) {
      return
    }
    let cancelled = false
    setLogs(undefined)
    setFailed(false)
    ChangelogBetween(game, update.nexusId, update.installed, update.version).then(
      (got) => {
        if (!cancelled) {
          setLogs(got ?? [])
        }
      },
      (e: unknown) => {
        if (!cancelled) {
          setFailed(true)
          reportUnexpected(e)
        }
      },
    )
    return () => {
      cancelled = true
    }
  }, [game, update.installed, update.nexusId, update.version])
  if (!update.nexusId) {
    return null
  }
  if (failed) {
    return null
  }
  if (logs === undefined) {
    return <Box role="status" aria-busy={true} sx={{ minHeight: 20 }} />
  }
  if (logs.length === 0) {
    return (
      <Typography
        sx={{ fontSize: 12, color: 'text.secondary' }}
      >{t`No changelog on Nexus`}</Typography>
    )
  }
  return (
    <Fold title={t`What's new`}>
      {logs.map((c) => (
        <Box key={c.version}>
          <Typography sx={{ fontSize: 13, fontWeight: 700 }}>{c.version}</Typography>
          {(c.notes ?? []).map((n) => (
            <Typography
              key={n}
              sx={{
                fontSize: 13,
                pl: 2,
                whiteSpace: 'pre-line',
                overflowWrap: 'anywhere',
                color: changelogNoteIsRisky(n) ? 'warning.main' : undefined,
              }}
            >
              {`• ${n}`}
            </Typography>
          ))}
        </Box>
      ))}
    </Fold>
  )
}
