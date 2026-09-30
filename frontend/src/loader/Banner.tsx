import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Check, Download } from 'lucide-react'
import { useEffect } from 'react'
import { useLoader } from './store.ts'

export function LoaderBanner({ game }: { game: string }) {
  const { t } = useLingui()
  const status = useLoader((s) => s.status)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const check = useLoader((s) => s.check)
  const install = useLoader((s) => s.install)
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
  const labels: Record<string, string> = {
    downloaded: t`Downloaded`,
    files: t`Files added`,
    launcher: t`Launcher replaced`,
    bundled: t`Bundled mods added`,
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
        <Box sx={{ display: 'flex', gap: 1.5, flexShrink: 0 }}>
          {steps.map((step) => (
            <Box
              key={step}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 0.5,
                fontSize: 13,
                color: 'text.secondary',
                whiteSpace: 'nowrap',
              }}
            >
              <Check size={14} />
              {labels[step] ?? step}
            </Box>
          ))}
        </Box>
      ) : (
        <Button
          variant="contained"
          size="small"
          startIcon={<Download size={16} />}
          onClick={() => install(game)}
          sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {action}
        </Button>
      )}
    </Box>
  )
}
