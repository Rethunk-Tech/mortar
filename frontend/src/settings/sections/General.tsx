import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select, Switch, TextField } from '@mui/material'
import { Download, Upload } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import {
  FirewallBlocked,
  FixFirewall,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/lan/service.ts'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  ExportSettings,
  PreviewImportSettings,
  SetKeepInTray,
  SetLanguage,
  SetLanPort,
  SetLanSharing,
  SetTipsSeen,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { TourAgainButton } from '../../firstrunTour/TourAgainButton.tsx'
import { availableLocales } from '../../i18n/locales.ts'
import { reportError, reportUnexpected } from '../../toasts/report.ts'
import { PrefKeys } from '../PrefRow.tsx'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { ImportSettingsDialog } from './DataImport.tsx'

const maxLanPort = 65_535
const defaultLanPort = 8080
const nowrap = { whiteSpace: 'nowrap' } as const

function useReportFailure() {
  const { t } = useLingui()
  return useCallback(reportError(t`Could not save that setting`), [])
}

function StartupAndWindow() {
  const { t } = useLingui()
  const reportFailure = useReportFailure()
  return (
    <SettingsSection title={t`Startup and window`}>
      <PrefKeys
        keys={['onPlay', 'startScreen', 'launchAtLogin', 'startMinimised', 'rememberWindow']}
      />
      <SettingRow
        label={t`Keep Mortar in the tray`}
        description={t`Closing the window hides Mortar instead of quitting.`}
      >
        <Switch
          checked={useSettings((s) => s.keepInTray)}
          onChange={(_, on) => SetKeepInTray(on).catch(reportFailure)}
        />
      </SettingRow>
    </SettingsSection>
  )
}

function Language() {
  const { t } = useLingui()
  const reportFailure = useReportFailure()
  const language = useSettings((s) => s.language)
  const single = availableLocales.length === 1
  return (
    <SettingsSection title={t`Language`}>
      <SettingRow
        label={t`Language`}
        {...(single ? { description: t`More languages are coming.` } : {})}
      >
        <Select
          size="small"
          value={language}
          disabled={single}
          displayEmpty={true}
          inputProps={{ 'aria-label': t`Language` }}
          onChange={(event) => {
            SetLanguage(event.target.value).catch(reportFailure)
          }}
          sx={{ minWidth: 180 }}
        >
          <MenuItem value="">{t`System default`}</MenuItem>
          {availableLocales.map((locale) => (
            <MenuItem key={locale} value={locale}>
              {new Intl.DisplayNames([locale], { type: 'language' }).of(locale) ?? locale}
            </MenuItem>
          ))}
        </Select>
      </SettingRow>
    </SettingsSection>
  )
}

function Sharing() {
  const { t } = useLingui()
  const reportFailure = useReportFailure()
  const [firewallBlocked, setFirewallBlocked] = useState(false)
  const [fixingFirewall, setFixingFirewall] = useState(false)
  const lanSharing = useSettings((s) => s.lanSharing)
  const lanPort = useSettings((s) => s.lanPort)
  const [portText, setPortText] = useState(String(lanPort))
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
    <SettingsSection title={t`Sharing`}>
      <SettingRow
        label={t`Share profiles on the local network`}
        description={t`Lets nearby Mortar users find this installation and exchange profile links.`}
      >
        <Switch checked={lanSharing} onChange={(_, on) => SetLanSharing(on).catch(reportFailure)} />
      </SettingRow>
      <PrefKeys
        keys={[
          'lanName',
          'lanAutoAcceptSameAccount',
          'shareIncludeDisabledMods',
          'shareIncludeFomodChoices',
          'shareIncludeNotes',
          'shareIncludeConfigFiles',
        ]}
      />
      {lanSharing ? (
        <SettingRow
          label={t`Automatic port`}
          description={t`Let the operating system choose a free port`}
        >
          <Switch
            checked={lanPort === 0}
            onChange={(_, on) => SetLanPort(on ? 0 : defaultLanPort).catch(reportFailure)}
          />
        </SettingRow>
      ) : null}
      {lanSharing && lanPort !== 0 ? (
        <SettingRow label={t`LAN port`} description={t`The port Mortar uses for local sharing`}>
          <TextField
            type="number"
            size="small"
            value={portText}
            slotProps={{
              htmlInput: { min: 1, max: maxLanPort, step: 1, 'aria-label': t`LAN port` },
            }}
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
      ) : null}
      {firewallBlocked ? (
        <SettingRow
          label={t`Windows Firewall blocks incoming sends to Mortar`}
          description={t`Allow Mortar through the firewall so nearby users can send profiles.`}
        >
          <Button
            variant="outlined"
            disabled={fixingFirewall}
            onClick={() => {
              setFixingFirewall(true)
              FixFirewall()
                .then(() => FirewallBlocked())
                .then(setFirewallBlocked)
                .catch(reportFailure)
                .finally(() => setFixingFirewall(false))
            }}
          >
            {fixingFirewall ? t`Fixing…` : t`Fix`}
          </Button>
        </SettingRow>
      ) : null}
    </SettingsSection>
  )
}

function Help() {
  const { t } = useLingui()
  const reportFailure = useReportFailure()
  return (
    <SettingsSection title={t`Help`}>
      <SettingRow label={t`Tips`} description={t`Show the first-run tips again`}>
        <Button
          variant="outlined"
          onClick={() => SetTipsSeen([]).catch(reportFailure)}
        >{t`Show again`}</Button>
      </SettingRow>
      <SettingRow
        label={t`Interface tour`}
        description={t`Walk through profiles, Play, mods, and the command palette`}
      >
        <TourAgainButton />
      </SettingRow>
    </SettingsSection>
  )
}

function SettingsFile() {
  const { t } = useLingui()
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null)
  return (
    <SettingsSection title={t`Settings file`}>
      <SettingRow
        label={t`Settings file`}
        description={t`Save these settings to a file, or load them on another computer.`}
      >
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="outlined"
            onClick={() => ExportSettings().catch(reportUnexpected)}
            startIcon={<Download size={16} />}
            sx={nowrap}
          >
            {t`Export…`}
          </Button>
          <Button
            variant="outlined"
            onClick={() => {
              PreviewImportSettings()
                .then((next) => {
                  if (next.raw) {
                    setImportPreview(next)
                  }
                })
                .catch(reportUnexpected)
            }}
            startIcon={<Upload size={16} />}
            sx={nowrap}
          >
            {t`Import…`}
          </Button>
        </Box>
      </SettingRow>
      <ImportSettingsDialog preview={importPreview} onClose={() => setImportPreview(null)} />
    </SettingsSection>
  )
}

export function General() {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <StartupAndWindow />
      <Language />
      <Sharing />
      <Help />
      <SettingsFile />
    </Box>
  )
}
