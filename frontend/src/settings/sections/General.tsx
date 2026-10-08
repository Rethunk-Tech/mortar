import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select, TextField, Typography } from '@mui/material'
import { Download, Upload } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import {
  FirewallBlocked,
  FixFirewall,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/lan/service.ts'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import {
  AntivirusProduct,
  ExportSettings,
  PickImportFile,
  PreviewImport,
  SetKeepInTray,
  SetLanguage,
  SetLanPort,
  SetLanSharing,
  SetTipsSeen,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { TourAgainButton } from '../../firstrunTour/TourAgainButton.tsx'
import { availableLocales } from '../../i18n/locales.ts'
import { PairedComputers } from '../../lan/PairedComputers.tsx'
import { useLoaded } from '../../shell/useLoaded.ts'
import { space } from '../../theme/density.ts'
import { reportError, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { ImportSettingsDialog } from './DataImport.tsx'

const maxLanPort = 65_535
const defaultLanPort = 8080

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
        <PrefSwitch
          checked={useSettings((s) => s.keepInTray)}
          onChange={(on) => SetKeepInTray(on).catch(reportFailure)}
          label={t`Keep Mortar in the tray`}
        />
      </SettingRow>
    </SettingsSection>
  )
}

function Antivirus() {
  const { t } = useLingui()
  const mode = useSettings((s) => s.antivirus)
  const socket = useSettings((s) => s.antivirusSocket)
  const command = useSettings((s) => s.antivirusCommand)
  // The product line follows the choices that decide which scanner answers.
  const { data: product } = useLoaded(
    () => AntivirusProduct(),
    [mode, socket, command],
    '',
    reportUnexpected,
  )
  return (
    <SettingsSection title={t`Antivirus`}>
      <PrefByKey prefKey="antivirus" hideTitle={true} />
      <SettingRow label={t`Scanner in use`}>
        <Typography sx={{ fontSize: 14 }}>{product || t`Checking…`}</Typography>
      </SettingRow>
      {mode === 'clamd' ? <PrefByKey prefKey="antivirusSocket" /> : null}
      {mode === 'command' ? <PrefByKey prefKey="antivirusCommand" /> : null}
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
        hideTitle={true}
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
  const [portError, setPortError] = useState(false)
  useEffect(() => {
    FirewallBlocked().then(setFirewallBlocked).catch(reportFailure)
  }, [reportFailure])
  useEffect(() => {
    setPortText(String(lanPort))
    setPortError(false)
  }, [lanPort])
  const portRange = t`Enter a number from ${{ min: 1 }} to ${{ max: maxLanPort }}`
  const savePort = () => {
    if (portText.trim() === '') {
      setPortError(true)
      return
    }
    const port = Number(portText)
    if (Number.isInteger(port) && port >= 1 && port <= maxLanPort) {
      setPortError(false)
      SetLanPort(port).catch(reportFailure)
    } else {
      setPortError(true)
    }
  }
  return (
    <SettingsSection title={t`Sharing`}>
      <SettingRow
        label={t`Share profiles on the local network`}
        description={t`Lets nearby Mortar users find this installation and exchange profile links.`}
      >
        <PrefSwitch
          checked={lanSharing}
          onChange={(on) => SetLanSharing(on).catch(reportFailure)}
          label={t`Share profiles on the local network`}
        />
      </SettingRow>
      <PrefKeys
        keys={[
          'lanName',
          'lanAutoAcceptPaired',
          'lanAllowAnyAddress',
          'shareIncludeDisabledMods',
          'shareIncludeFomodChoices',
          'shareIncludeNotes',
          'shareIncludeConfigFiles',
          'shareIncludeProblemChoices',
        ]}
      />
      {lanSharing ? <PairedComputers /> : null}
      {lanSharing ? (
        <SettingRow
          label={t`Automatic port`}
          description={t`Let the operating system choose a free port`}
        >
          <PrefSwitch
            checked={lanPort === 0}
            onChange={(on) => SetLanPort(on ? 0 : defaultLanPort).catch(reportFailure)}
            label={t`Automatic port`}
          />
        </SettingRow>
      ) : null}
      {lanSharing && lanPort !== 0 ? (
        <SettingRow label={t`LAN port`} description={t`The port Mortar uses for local sharing`}>
          <TextField
            type="number"
            size="small"
            value={portText}
            error={portError}
            helperText={portError ? portRange : undefined}
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
            {t`Allow through firewall`}
          </Button>
        </SettingRow>
      ) : null}
    </SettingsSection>
  )
}

function Help() {
  const { t } = useLingui()
  const reportFailure = useReportFailure()
  const push = useToasts((s) => s.push)
  return (
    <SettingsSection title={t`Help`}>
      <SettingRow label={t`Tips`} description={t`Show the first-run tips again`}>
        <Button
          variant="outlined"
          onClick={() =>
            SetTipsSeen([])
              .then(() => push({ kind: 'success', title: t`Tips will show again` }))
              .catch(reportFailure)
          }
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
  const [importPath, setImportPath] = useState('')
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null)
  return (
    <SettingsSection title={t`Settings file`}>
      <SettingRow
        label={t`Settings file`}
        hideTitle={true}
        description={t`Save these settings to a file, or load them on another computer.`}
      >
        <Box sx={{ display: 'flex', gap: space.gap }}>
          <Button
            variant="outlined"
            onClick={() => ExportSettings().catch(reportUnexpected)}
            startIcon={<Download size={16} />}
          >
            {t`Export…`}
          </Button>
          <Button
            variant="outlined"
            onClick={() => {
              PickImportFile()
                .then((path) =>
                  path ? PreviewImport(path).then((next) => ({ path, next })) : undefined,
                )
                .then((picked) => {
                  if (picked) {
                    setImportPath(picked.path)
                    setImportPreview(picked.next)
                  }
                })
                .catch(reportUnexpected)
            }}
            startIcon={<Upload size={16} />}
          >
            {t`Import…`}
          </Button>
        </Box>
      </SettingRow>
      <ImportSettingsDialog
        path={importPath}
        preview={importPreview}
        onClose={() => setImportPreview(null)}
      />
    </SettingsSection>
  )
}

export function General() {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
      <StartupAndWindow />
      <Language />
      <Antivirus />
      <Sharing />
      <Help />
      <SettingsFile />
    </Box>
  )
}
