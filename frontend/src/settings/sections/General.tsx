import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, Select, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Download, Upload } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import {
  FirewallBlocked,
  FixFirewall,
  NetworkIsPublic,
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
import { WEBLATE, WINDOWS } from '../../brand/productNames.ts'
import { TRANSLATE_PROJECT } from '../../brand/translateProject.ts'
import { TourAgainButton } from '../../firstrunTour/TourAgainButton.tsx'
import { localeLabel, pickerLocales } from '../../i18n/locales.ts'
import { useReasonText } from '../../i18n/reasonText.ts'
import { PairedComputers } from '../../lan/PairedComputers.tsx'
import { openPage } from '../../mods/menu.ts'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefNumber, PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { settingSaveFailed } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useSettingsRelevant } from '../useSettingsSearch.ts'
import { CodeEditor } from './CodeEditor.tsx'
import { ImportSettingsDialog } from './DataImport.tsx'

const maxLanPort = 65_535
const defaultLanPort = 8080

function StartupAndWindow() {
  const { t } = useLingui()
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
          onChange={(on) => SetKeepInTray(on).catch(settingSaveFailed)}
          label={t`Keep Mortar in the tray`}
        />
      </SettingRow>
    </SettingsSection>
  )
}

// Its own component, so the answer arriving re-renders this line and not the section.
function ScannerName() {
  const { t } = useLingui()
  const mode = useSettings((s) => s.antivirus)
  const socket = useSettings((s) => s.antivirusSocket)
  const command = useSettings((s) => s.antivirusCommand)
  // The product line follows the choices that decide which scanner answers.
  const { data: product } = useLoaded(
    () => AntivirusProduct(),
    [mode, socket, command],
    { text: '' },
    reportUnexpected,
  )
  const reasonText = useReasonText()
  const line =
    product.text === '' ? t`Checking…` : reasonText(product.code, product.params, product.text)
  return <Typography sx={{ fontSize: 'medium.fontSize' }}>{line}</Typography>
}

function Antivirus() {
  const { t } = useLingui()
  const mode = useSettings((s) => s.antivirus)
  return (
    <SettingsSection title={t`Antivirus`}>
      <PrefByKey prefKey="antivirus" hideTitle={true} />
      <SettingRow label={t`Scanner in use`}>
        <ScannerName />
      </SettingRow>
      {mode === 'clamd' ? <PrefByKey prefKey="antivirusSocket" /> : null}
      {mode === 'command' ? <PrefByKey prefKey="antivirusCommand" /> : null}
    </SettingsSection>
  )
}

function Language() {
  const { t } = useLingui()
  const language = useSettings((s) => s.language)
  return (
    <SettingsSection title={t`Language`}>
      <SettingRow
        label={t`Display language`}
        description={t`Mortar follows the system language unless you pick one`}
      >
        <Select
          size="small"
          value={language}
          displayEmpty={true}
          slotProps={{ input: { 'aria-label': t`Language` } }}
          onChange={(event) => {
            SetLanguage(event.target.value).catch(settingSaveFailed)
          }}
          sx={{ minWidth: 180 }}
        >
          <MenuItem value="">{t`System default`}</MenuItem>
          {pickerLocales(language).map((locale) => (
            <MenuItem key={locale} value={locale}>
              {localeLabel(locale)}
            </MenuItem>
          ))}
        </Select>
      </SettingRow>
      <SettingRow
        label={t`Help translate Mortar`}
        description={t`Players translate Mortar on ${{ app: WEBLATE }}, and any language can be added there.`}
      >
        <Button variant="outlined" onClick={() => openPage(TRANSLATE_PROJECT)}>
          {t`Open ${{ name: WEBLATE }}`}
        </Button>
      </SettingRow>
    </SettingsSection>
  )
}

// The LAN rows only mean something while sharing is on; what to share also sets the file and link defaults.
function SharingPrefs({ on }: { on: boolean }) {
  const { t } = useLingui()
  const first = t`Enable Share profiles on the local network first.`
  return (
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
        'shareIncludeLibraries',
      ]}
      disabledBy={
        on ? {} : { lanName: first, lanAutoAcceptPaired: first, lanAllowAnyAddress: first }
      }
    />
  )
}

function Sharing() {
  const { t } = useLingui()
  const [firewallBlocked, setFirewallBlocked] = useState(false)
  const [fixingFirewall, setFixingFirewall] = useState(false)
  const [networkPublic, setNetworkPublic] = useState(false)
  const allowAny = useSettings((s) => s.lanAllowAnyAddress)
  const lanSharing = useSettings((s) => s.lanSharing)
  const lanPort = useSettings((s) => s.lanPort)
  const loadStatus = useCallback(() => {
    FirewallBlocked().then(setFirewallBlocked).catch(reportUnexpected)
    NetworkIsPublic().then(setNetworkPublic).catch(reportUnexpected)
  }, [])
  // The firewall rules depend on both settings, so a change to either rereads the status; with sharing off, or a search
  // these rows do not match, there is nothing to show.
  const publicLabel = t`This network is set to Public in ${{ os: WINDOWS }}`
  const publicText = t`Nearby computers cannot reach Mortar. Set the network to Private in ${{ os: WINDOWS }} network settings.`
  const firewallLabel = t`${{ os: WINDOWS }} Firewall blocks incoming sends to Mortar`
  const firewallText = t`Allow Mortar through the firewall so nearby users can send profiles.`
  const relevant = useSettingsRelevant(publicLabel, publicText, firewallLabel, firewallText)
  const statusKey = `${lanSharing && relevant}:${allowAny}`
  useEffect(() => {
    if (!statusKey.startsWith('true')) {
      setFirewallBlocked(false)
      setNetworkPublic(false)
      return
    }
    loadStatus()
  }, [statusKey, loadStatus])
  return (
    <SettingsSection title={t`Sharing`}>
      <SettingRow
        label={t`Share profiles on the local network`}
        description={t`Lets nearby Mortar users find this installation and exchange profile links.`}
      >
        <PrefSwitch
          checked={lanSharing}
          onChange={(on) => SetLanSharing(on).catch(settingSaveFailed)}
          label={t`Share profiles on the local network`}
        />
      </SettingRow>
      <SharingPrefs on={lanSharing} />
      {lanSharing ? <PairedComputers /> : null}
      {lanSharing ? (
        <SettingRow
          label={t`Automatic port`}
          description={t`Let the operating system choose a free port`}
        >
          <PrefSwitch
            checked={lanPort === 0}
            onChange={(on) => SetLanPort(on ? 0 : defaultLanPort).catch(settingSaveFailed)}
            label={t`Automatic port`}
          />
        </SettingRow>
      ) : null}
      {lanSharing && lanPort !== 0 ? (
        <SettingRow label={t`LAN port`} description={t`The port Mortar uses for local sharing`}>
          <PrefNumber
            value={lanPort}
            min={1}
            max={maxLanPort}
            onCommit={SetLanPort}
            label={t`LAN port`}
            width={140}
          />
        </SettingRow>
      ) : null}
      {lanSharing && networkPublic && !allowAny ? (
        <SettingRow label={publicLabel} description={publicText}>
          <span />
        </SettingRow>
      ) : null}
      {lanSharing && firewallBlocked ? (
        <SettingRow label={firewallLabel} description={firewallText}>
          <Button
            variant="outlined"
            disabled={fixingFirewall}
            onClick={() => {
              setFixingFirewall(true)
              FixFirewall()
                .then(loadStatus)
                .catch(reportUnexpected)
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
  const push = useToasts((s) => s.push)
  return (
    <SettingsSection title={t`Help`}>
      <SettingRow label={t`Tips`} description={t`Show the first-run tips again`}>
        <Button
          variant="outlined"
          onClick={() =>
            SetTipsSeen([])
              .then(() => push({ kind: 'success', title: t`Tips will show again` }))
              .catch(settingSaveFailed)
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
      <CodeEditor />
      <Antivirus />
      <Sharing />
      <Help />
      <SettingsFile />
    </Box>
  )
}
