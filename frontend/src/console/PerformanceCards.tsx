import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import type { StartupReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import { space } from '../theme/density.ts'
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
          fontSize: 12,
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
          value={
            saved > 0
              ? plural(
                  { count: saved },
                  { one: '# measurement saved', other: '# measurements saved' },
                )
              : t`Not measured yet`
          }
          on={view === 'inGame'}
          onClick={() => onChoose('inGame')}
        >
          {running
            ? t`The game is running: measure it now`
            : t`Play this profile, then time each mod per frame`}
        </Card>
      ) : null}
    </Box>
  )
}
