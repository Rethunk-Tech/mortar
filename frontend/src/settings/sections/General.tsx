import { useLingui } from '@lingui/react/macro'
import { Box, Button, FormControlLabel, Switch, TextField } from '@mui/material'
import {
  SetEnableModsWhenInstalled,
  SetKeepInTray,
  SetLanPort,
  SetLanSharing,
  SetTipsSeen,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { errorText } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { useSettings } from '../store.ts'

const maxLanPort = 65_535

function KeepInTraySwitch() {
  const { t } = useLingui()
  const keepInTray = useSettings((s) => s.keepInTray)
  const push = useToasts((s) => s.push)
  const reportFailure = (err: unknown) => {
    const body = errorText(err)
    push({ kind: 'error', title: t`Couldn't save that setting`, ...(body ? { body } : {}) })
  }
  return (
    <FormControlLabel
      sx={{ m: 0, alignItems: 'flex-start' }}
      control={
        <Switch
          checked={keepInTray}
          onChange={(_, on) => {
            SetKeepInTray(on).catch(reportFailure)
          }}
        />
      }
      label={
        <Box>
          <Box component="span" sx={{ display: 'block', fontSize: 14 }}>
            {t`Keep Mortar in the tray`}
          </Box>
          <Box
            component="span"
            sx={{ display: 'block', fontSize: 13, color: 'rgba(225,225,230,0.95)' }}
          >
            {t`Closing the window hides Mortar instead of quitting. On GNOME you may need the AppIndicator extension to see the tray icon.`}
          </Box>
        </Box>
      }
    />
  )
}

export function General() {
  const { t } = useLingui()
  const enableModsWhenInstalled = useSettings((s) => s.enableModsWhenInstalled)
  const lanSharing = useSettings((s) => s.lanSharing)
  const lanPort = useSettings((s) => s.lanPort)
  const push = useToasts((s) => s.push)
  const reportFailure = (err: unknown) => {
    const body = errorText(err)
    push({ kind: 'error', title: t`Couldn't save that setting`, ...(body ? { body } : {}) })
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Window`}</Box>
      <KeepInTraySwitch />
      <Button
        variant="outlined"
        onClick={() => {
          SetTipsSeen([]).catch(reportFailure)
        }}
        sx={{ alignSelf: 'flex-start', whiteSpace: 'nowrap' }}
      >
        {t`Show tips again`}
      </Button>
      <Box sx={{ fontSize: 14, fontWeight: 600, pt: 1 }}>{t`Sharing`}</Box>
      <FormControlLabel
        sx={{ m: 0, alignItems: 'flex-start' }}
        control={
          <Switch
            checked={lanSharing}
            onChange={(_, on) => {
              SetLanSharing(on).catch(reportFailure)
            }}
          />
        }
        label={
          <Box>
            <Box component="span" sx={{ display: 'block', fontSize: 14 }}>
              {t`Share profiles on the local network`}
            </Box>
            <Box
              component="span"
              sx={{ display: 'block', fontSize: 13, color: 'rgba(225,225,230,0.95)' }}
            >
              {t`Lets nearby Mortar users find this installation and exchange profile links.`}
            </Box>
          </Box>
        }
      />
      <TextField
        label={t`LAN port`}
        type="number"
        size="small"
        value={lanPort}
        slotProps={{ htmlInput: { min: 0, max: maxLanPort, step: 1 } }}
        helperText={t`Use 0 to let the operating system choose a port.`}
        onChange={(event) => {
          const port = Number(event.target.value)
          if (Number.isInteger(port) && port >= 0 && port <= maxLanPort) {
            SetLanPort(port).catch(reportFailure)
          }
        }}
        sx={{ maxWidth: 240 }}
      />
      <Box sx={{ fontSize: 14, fontWeight: 600, pt: 1 }}>{t`Mods`}</Box>
      <FormControlLabel
        sx={{ m: 0, alignItems: 'center' }}
        control={
          <Switch
            checked={enableModsWhenInstalled !== false}
            onChange={(_, on) => {
              SetEnableModsWhenInstalled(on).catch(reportFailure)
            }}
          />
        }
        label={t`Enable mods when installed`}
      />
    </Box>
  )
}
