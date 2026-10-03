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
  SetListSort,
  SetTipsSeen,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { errorText } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect } from '../PrefControls.tsx'
import { PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const maxLanPort = 65_535
const defaultLanPort = 8080

function WindowLaunch() {
  return (
    <PrefKeys
      keys={[
        'onPlay',
        'startScreen',
        'defaultLaunchMethod',
        'showSmapiConsole',
        'launchAtLogin',
        'startMinimised',
        'rememberWindow',
      ]}
    />
  )
}

function LanIdentity() {
  return (
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
  )
}

function ModsPrefs() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  const sort = `${useSettings((s) => s.listSortColumn) || 'name'}:${useSettings((s) => s.listSortDir) || 'asc'}`
  return (
    <SettingsSection
      title={t`Mods`}
      prefKeys={[
        'defaultModsView',
        'gridCardSize',
        'showAuthorOnCards',
        'enableRequirements',
        'missingRequirements',
        'reuseFomodChoices',
        'listGroupBy',
        'listSortColumn',
        'listSortDir',
        'confirmRemovals',
        'cosmeticConflicts',
        'conflictScanDepth',
        'backgroundBadgeChecks',
        'profileOrder',
        'sidebarBadges',
      ]}
    >
      <PrefKeys
        keys={[
          'defaultModsView',
          'gridCardSize',
          'showAuthorOnCards',
          'enableRequirements',
          'missingRequirements',
          'reuseFomodChoices',
          'listGroupBy',
        ]}
      />
      <SettingRow label={t`Default sort`}>
        <PrefSelect
          value={sort}
          onChange={(v) => {
            const [column, dir] = v.split(':')
            persist(() => SetListSort(column ?? 'name', dir ?? 'asc'), push, fail)
          }}
          options={[
            { value: 'name:asc', label: t`Name A–Z` },
            { value: 'name:desc', label: t`Name Z–A` },
            { value: 'version:asc', label: t`Version` },
            { value: 'author:asc', label: t`Author` },
          ]}
        />
      </SettingRow>
      <PrefKeys
        keys={[
          'confirmRemovals',
          'cosmeticConflicts',
          'conflictScanDepth',
          'backgroundBadgeChecks',
          'profileOrder',
          'sidebarBadges',
        ]}
      />
    </SettingsSection>
  )
}

function DisplayAndNotices() {
  const { t } = useLingui()
  return (
    <>
      <SettingsSection
        title={t`Display`}
        prefKeys={['dates', 'density', 'reduceMotion', 'profileHero']}
      >
        <PrefKeys keys={['dates', 'density', 'reduceMotion', 'profileHero']} />
      </SettingsSection>
      <SettingsSection
        title={t`Notifications`}
        prefKeys={['notifyDownloadFinished', 'notifyDownloadFailed', 'notifyRunCrashed']}
      >
        <PrefKeys keys={['notifyDownloadFinished', 'notifyDownloadFailed', 'notifyRunCrashed']} />
      </SettingsSection>
    </>
  )
}

export function General() {
  const { t } = useLingui()
  const [firewallBlocked, setFirewallBlocked] = useState(false)
  const [fixingFirewall, setFixingFirewall] = useState(false)
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
      <SettingsSection
        title={t`Window`}
        prefKeys={['onPlay', 'startScreen', 'defaultLaunchMethod', 'showSmapiConsole']}
      >
        <WindowLaunch />
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
      <SettingsSection title={t`Sharing`} prefKeys={['lanName', 'lanAutoAcceptSameAccount']}>
        <SettingRow
          label={t`Share profiles on the local network`}
          description={t`Lets nearby Mortar users find this installation and exchange profile links.`}
        >
          <Switch
            checked={lanSharing}
            onChange={(_, on) => SetLanSharing(on).catch(reportFailure)}
          />
        </SettingRow>
        <LanIdentity />
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
      <SettingsSection title={t`Stardew Valley`}>
        <ModsPrefs />
      </SettingsSection>
      <DisplayAndNotices />
    </Box>
  )
}
