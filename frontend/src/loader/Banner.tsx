import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Download } from 'lucide-react'
import { useEffect } from 'react'
import { useGameInfo } from '../games/info.ts'
import { useGameBusy } from '../launch/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
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
  const loader = { name: useGameInfo(game)?.loader ?? '' }
  // The loader's files are in use while the game starts or runs.
  const playing = useGameBusy(game)
  useEffect(() => {
    check(game)
  }, [game, check])
  if (!status || (!installing && status.installed && !status.updateAvailable)) {
    return null
  }
  let message = t`${loader.name} is not installed`
  let action = t`Install`
  if (status.broken) {
    message = t`A game update replaced ${loader.name}'s launcher`
    action = t`Reinstall`
  } else if (status.installed) {
    message = t`${loader.name} ${status.latest} is available`
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
        bgcolor: 'var(--mortar-panel)',
        borderLeft: '4px solid',
        borderLeftColor: status.installed && !status.broken ? 'info.main' : 'warning.main',
      }}
    >
      <Typography noWrap={true} title={message} sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
        {message}
      </Typography>
      {installing ? (
        <InstallSteps steps={steps} />
      ) : (
        <DisabledReason
          title={t`Stop the game to change ${loader.name}.`}
          disabled={pending || playing}
        >
          <Button
            variant="contained"
            size="small"
            startIcon={<Download size={16} />}
            disabled={pending || playing}
            onClick={() => install(game)}
            sx={{ flexShrink: 0 }}
          >
            {action}
          </Button>
        </DisabledReason>
      )}
    </Box>
  )
}
