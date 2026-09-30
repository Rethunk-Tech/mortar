import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Download } from 'lucide-react'
import { useEffect } from 'react'
import { State } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { useLaunch } from '../launch/store.ts'
import { InstallSteps } from './InstallSteps.tsx'
import { useLoader } from './store.ts'

export function LoaderBanner({ game }: { game: string }) {
  const { t } = useLingui()
  const status = useLoader((s) => s.status)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const check = useLoader((s) => s.check)
  const install = useLoader((s) => s.install)
  const pending = useLoader((s) => s.pending)
  // SMAPI's files are in use while the game starts or runs.
  const playing = useLaunch(
    (s) =>
      s.starting ||
      (s.status?.game === game &&
        (s.status.state === State.Launching || s.status.state === State.Running)),
  )
  useEffect(() => {
    check(game)
  }, [game, check])
  if (!status || (!installing && status.installed && !status.updateAvailable)) {
    return null
  }
  let message = t`SMAPI is not installed`
  let action = t`Install`
  if (status.broken) {
    message = t`A game update replaced SMAPI's launcher`
    action = t`Reinstall`
  } else if (status.installed) {
    message = t`SMAPI ${status.latest} is available`
    action = t`Update`
  }
  return (
    <Box
      role="status"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 2,
        px: 2,
        py: 1,
        flexShrink: 0,
        bgcolor: 'rgba(40,40,48,0.6)',
        borderLeft: '4px solid',
        borderLeftColor: status.installed && !status.broken ? 'info.main' : 'warning.main',
      }}
    >
      <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
        {message}
      </Typography>
      {installing ? (
        <InstallSteps steps={steps} />
      ) : (
        <Button
          variant="contained"
          size="small"
          startIcon={<Download size={16} />}
          disabled={pending || playing}
          onClick={() => install(game)}
          sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {action}
        </Button>
      )}
    </Box>
  )
}
