import { useLingui } from '@lingui/react/macro'
import { Box, Button, Switch, TextField } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import {
  FirewallBlocked,
  FixFirewall,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/lan/service.ts'
import {
  SetEnableModsWhenInstalled,
  SetKeepInTray,
  SetLanPort,
  SetLanSharing,
  SetTipsSeen,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { errorText } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const maxLanPort = 65_535
const defaultLanPort = 8080

export function General() {
  const { t } = useLingui()
  const [firewallBlocked, setFirewallBlocked] = useState(false)
  const enableModsWhenInstalled = useSettings((s) => s.enableModsWhenInstalled)
  const lanSharing = useSettings((s) => s.lanSharing)
  const lanPort = useSettings((s) => s.lanPort)
  const [portText, setPortText] = useState(String(lanPort))
  const push = useToasts((s) => s.push)
  const reportFailure = useCallback(
    (err: unknown) => {
      const body = errorText(err)
      push({ kind: 'error', title: t`Couldn't save that setting`, ...(body ? { body } : {}) })
    },
    [push, t],
  )
  useEffect(() => {
    FirewallBlocked().then(setFirewallBlocked).catch(reportFailure)
  }, [reportFailure])
  useEffect(() => {
    setPortText(String(lanPort))
  }, [lanPort])
  const savePort = () => {
    if (portText.trim() === '') {
      setPortText(String(lanPort))
      return
    }
    const port = Number(portText)
    if (Number.isInteger(port) && port >= 0 && port <= maxLanPort) {
      SetLanPort(port).catch(reportFailure)
    } else {
      setPortText(String(lanPort))
    }
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <SettingsSection title={t`Window`}>
        <SettingRow
          label={t`Keep Mortar in the tray`}
          description={t`Closing the window hides Mortar instead of quitting.`}
        >
          <Switch
            checked={useSettings((s) => s.keepInTray)}
            onChange={(_, on) => SetKeepInTray(on).catch(reportFailure)}
          />
        </SettingRow>
        <SettingRow label={t`Tips`} description={t`Show the first-run tips again`}>
          <Button
            variant="outlined"
            onClick={() => SetTipsSeen([]).catch(reportFailure)}
          >{t`Show again`}</Button>
        </SettingRow>
      </SettingsSection>
      <SettingsSection title={t`Sharing`}>
        <SettingRow
          label={t`Share profiles on the local network`}
          description={t`Lets nearby Mortar users find this installation and exchange profile links.`}
        >
          <Switch
            checked={lanSharing}
            onChange={(_, on) => SetLanSharing(on).catch(reportFailure)}
          />
        </SettingRow>
        {lanSharing ? (
          <>
            <SettingRow
              label={t`Automatic port`}
              description={t`Let the operating system choose a free port`}
            >
              <Switch
                checked={lanPort === 0}
                onChange={(_, on) => SetLanPort(on ? 0 : defaultLanPort).catch(reportFailure)}
              />
            </SettingRow>
            {lanPort === 0 ? null : (
              <SettingRow
                label={t`LAN port`}
                description={t`The port Mortar uses for local sharing`}
              >
                <TextField
                  type="number"
                  size="small"
                  value={portText}
                  slotProps={{ htmlInput: { min: 1, max: maxLanPort, step: 1 } }}
                  onChange={(event) => setPortText(event.target.value)}
                  onBlur={savePort}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                      savePort()
                    }
                  }}
                  sx={{ width: 140 }}
                />
              </SettingRow>
            )}
          </>
        ) : null}
        {firewallBlocked === true && (
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1,
              color: 'rgba(255,220,170,0.95)',
              fontSize: 13,
            }}
          >
            <Box sx={{ flex: 1 }}>{t`Windows Firewall blocks incoming sends to Mortar`}</Box>
            <Button
              size="small"
              variant="outlined"
              onClick={() => {
                FixFirewall()
                  .then(() => FirewallBlocked())
                  .then(setFirewallBlocked)
                  .catch(reportFailure)
              }}
            >
              {t`Fix`}
            </Button>
          </Box>
        )}
      </SettingsSection>
      <SettingsSection title={t`Mods`}>
        <SettingRow label={t`Enable mods when installed`}>
          <Switch
            checked={enableModsWhenInstalled !== false}
            onChange={(_, on) => SetEnableModsWhenInstalled(on).catch(reportFailure)}
          />
        </SettingRow>
      </SettingsSection>
    </Box>
  )
}
