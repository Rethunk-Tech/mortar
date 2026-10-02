import { useLingui } from '@lingui/react/macro'
import { Box, Button, Menu, MenuItem } from '@mui/material'
import { ChevronDown, History } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Outcome } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import type { Run } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { Runs } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useConsole } from './store.ts'

export function RunsPicker({ game }: { game: string }) {
  const { t } = useLingui()
  const profile = useProfiles((s) => s.openId)
  const viewingRun = useConsole((s) => s.viewingRun)
  const viewRun = useConsole((s) => s.viewRun)
  const crashId = useLaunch((s) => s.crash?.runId ?? '')
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [runs, setRuns] = useState<Run[]>([])
  useEffect(() => {
    if (!profile) {
      setRuns([])
      return
    }
    let live = true
    const token = crashId
    Runs(game, profile).then(
      (list) => {
        if (live && token === crashId) {
          setRuns(list ?? [])
        }
      },
      () => {
        if (live) {
          setRuns([])
        }
      },
    )
    return () => {
      live = false
    }
  }, [game, profile, crashId])
  const outcome = (o: string) => {
    if (o === Outcome.OutcomeCrashed) {
      return t`Crashed`
    }
    if (o === Outcome.OutcomeFailed) {
      return t`Failed`
    }
    return t`Ran`
  }
  const label = (r: Run) => {
    const when = r.started ? formatWhen(r.started, { withTime: true }) : r.id
    return `${when} · ${outcome(r.outcome)}`
  }
  const selected = runs.find((r) => r.id === viewingRun)
  let button = t`This session`
  if (viewingRun !== '') {
    button = selected ? label(selected) : t`Past run`
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<History size={14} />}
        endIcon={<ChevronDown size={12} />}
        aria-label={t`Runs: ${button}`}
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{ height: 34, borderColor: 'rgba(255,255,255,0.2)', color: '#ffffff' }}
      >
        {button}
      </Button>
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        slotProps={{ paper: { sx: { maxHeight: 360, minWidth: 280 } } }}
      >
        <MenuItem
          dense={true}
          selected={viewingRun === ''}
          onClick={() => {
            viewRun(game, profile, '')
            setAnchor(null)
          }}
        >
          {t`This session`}
        </MenuItem>
        {runs.length === 0 ? <MenuItem disabled={true}>{t`No recorded runs yet`}</MenuItem> : null}
        {runs.map((r) => (
          <MenuItem
            key={r.id}
            dense={true}
            selected={viewingRun === r.id}
            onClick={() => {
              viewRun(game, profile, r.id)
              setAnchor(null)
            }}
          >
            {label(r)}
          </MenuItem>
        ))}
      </Menu>
    </Box>
  )
}
