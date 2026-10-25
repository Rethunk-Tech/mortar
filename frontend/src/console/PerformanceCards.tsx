import { i18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { space } from '../theme/density.ts'
import { usePerfLive } from './perfLive.ts'
import { useDuration, useSmapiStartup } from './startupHooks.ts'
import { modsMs } from './startupView.ts'

export type PerformanceView = 'startup' | 'inGame'

function Card({
  label,
  value,
  children,
  on,
  onClick,
}: {
  label: string
  value: string
  children: ReactNode
  on: boolean
  onClick: (() => void) | undefined
}) {
  return (
    <ButtonBase
      aria-pressed={onClick ? on : undefined}
      disabled={!onClick}
      onClick={onClick}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'flex-start',
        gap: 0.5,
        px: space.pad,
        py: 1.5,
        textAlign: 'left',
        borderRadius: '10px',
        bgcolor: 'var(--mortar-overlay-30)',
        border: '1px solid',
        borderColor: on ? 'primary.main' : 'var(--mortar-hairline-12)',
        boxShadow: (theme) => (on ? `inset 0 0 0 1px ${theme.palette.primary.main}` : 'none'),
        '&.Mui-disabled': { color: 'inherit' },
      }}
    >
      <Typography
        component="span"
        sx={{
          fontSize: 'small.fontSize',
          letterSpacing: '0.06em',
          textTransform: 'uppercase',
          color: 'text.secondary',
        }}
      >
        {label}
      </Typography>
      <Typography component="span" sx={{ fontSize: 22, fontWeight: 700 }}>
        {value}
      </Typography>
      <Typography component="span" variant="body2" sx={{ color: 'text.secondary' }}>
        {children}
      </Typography>
    </ButtonBase>
  )
}

function inGameValue(measuring: boolean, stopped: string, saved: number): string {
  if (measuring) {
    return i18n._(msg`Measuring`)
  }
  if (stopped) {
    return i18n._(msg`Measuring stopped`)
  }
  return saved > 0
    ? i18n._(
        msg`${plural({ count: saved }, { one: '# measurement saved', other: '# measurements saved' })}`,
      )
    : i18n._(msg`Not measured yet`)
}

function inGameNote(measuring: boolean, stopped: string, top: string, running: boolean): string {
  if (measuring) {
    return top
      ? i18n._(msg`Live: ${{ top }} takes the most time per frame`)
      : i18n._(msg`Waiting for the first results`)
  }
  if (stopped) {
    return stopped
  }
  return running
    ? i18n._(msg`The game is running: measure it now`)
    : i18n._(msg`Play this profile, then time each mod per frame`)
}

// The page's two views as cards that state their own verdict, so the choice between them is also the summary.
export function PerformanceCards({
  view,
  onChoose,
  report,
  calm,
  inGame,
  saved,
  running,
}: {
  view: PerformanceView
  onChoose: (view: PerformanceView) => void
  report: StartupReport | undefined
  calm: boolean
  inGame: boolean
  saved: number
  running: boolean
}) {
  const { t } = useLingui()
  const duration = useDuration()
  const smapi = useSmapiStartup()
  const measuring = usePerfLive((s) => running && s.measuring)
  const stopped = usePerfLive((s) => (running ? s.reason : ''))
  const top = usePerfLive((s) => s.top)
  let value = t`Not measured yet`
  let note = t`Play this profile to time how long each mod adds to the start`
  if (report) {
    const total = duration(report.phases.titleScreen)
    const mods = duration(modsMs(report))
    value = smapi ? t`${total} to the title screen` : t`${total} to the main menu`
    note = calm ? t`Mods add ${mods}. Nothing stands out in this start.` : t`Mods add ${mods}.`
  }
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: inGame ? '1fr 1fr' : '1fr',
        gap: space.gap,
      }}
    >
      <Card
        label={t`Startup`}
        value={value}
        on={inGame && view === 'startup'}
        onClick={inGame ? () => onChoose('startup') : undefined}
      >
        {note}
      </Card>
      {inGame ? (
        <Card
          label={t`In game`}
          value={inGameValue(measuring, stopped, saved)}
          on={view === 'inGame'}
          onClick={() => onChoose('inGame')}
        >
          {inGameNote(measuring, stopped, top, running)}
        </Card>
      ) : null}
    </Box>
  )
}
