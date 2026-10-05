import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ListItemText, Menu, MenuItem } from '@mui/material'
import { ChevronDown, History } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Run } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useOutcomeLabel } from './outcome.ts'
import { runExitText } from './runExit.ts'
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
  const outcome = useOutcomeLabel()
  const label = (r: Run) => {
    const when = r.started ? formatWhen(r.started, { withTime: true }) : r.id
    const preset = r.preset ? ` · ${r.preset}` : ''
    return `${when} · ${outcome(r.outcome)}${preset}`
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
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{ height: 34, borderColor: 'var(--mortar-hairline-20)', color: 'var(--mortar-ink)' }}
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
          role="menuitemradio"
          aria-checked={viewingRun === ''}
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
            role="menuitemradio"
            aria-checked={viewingRun === r.id}
            selected={viewingRun === r.id}
            title={runExitText(r.exit)}
            onClick={() => {
              viewRun(game, profile, r.id)
              setAnchor(null)
            }}
          >
            <ListItemText
              primary={label(r)}
              secondary={
                r.unclassified
                  ? t`${plural(r.unclassified, { one: '# log line Mortar could not classify', other: '# log lines Mortar could not classify' })}`
                  : undefined
              }
            />
          </MenuItem>
        ))}
      </Menu>
    </Box>
  )
}
