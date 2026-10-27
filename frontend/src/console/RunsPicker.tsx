import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Menu } from '@mui/material'
import { ChevronDown, History } from 'lucide-react'
import { useState } from 'react'
import type { Run } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useLaunch } from '../launch/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuCheck, MenuNote } from '../shell/MenuAction.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { listRuns } from './listRuns.ts'
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
  const { data: runs } = useLoaded<Run[]>(
    profile ? () => listRuns(game, profile) : null,
    [game, profile, crashId],
    [],
  )
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
        sx={{
          height: space.control,
          borderColor: 'var(--mortar-hairline-20)',
          color: 'var(--mortar-ink)',
        }}
      >
        {button}
      </Button>
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        slotProps={{ paper: { sx: { maxHeight: 360, minWidth: 280 } } }}
      >
        <MenuCheck
          radio={true}
          checked={viewingRun === ''}
          label={t`This session`}
          onClick={() => {
            viewRun(game, profile, '')
            setAnchor(null)
          }}
        />
        {runs.length === 0 ? <MenuNote>{t`No recorded runs yet`}</MenuNote> : null}
        {runs.map((r) => (
          <MenuCheck
            key={r.id}
            radio={true}
            checked={viewingRun === r.id}
            title={r.error || runExitText(r.exit)}
            label={label(r)}
            detail={
              r.error ||
              (r.unclassified
                ? plural(
                    { count: r.unclassified },
                    {
                      one: '# log line Mortar could not classify',
                      other: '# log lines Mortar could not classify',
                    },
                  )
                : undefined)
            }
            onClick={() => {
              viewRun(game, profile, r.id)
              setAnchor(null)
            }}
          />
        ))}
      </Menu>
    </Box>
  )
}
