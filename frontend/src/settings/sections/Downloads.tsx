import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link } from '@mui/material'
import { useEffect, useState } from 'react'
import type { GameInfo } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { List as ListGames } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { List as ListProfiles } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  SetByKey,
  SetNexusPreferredDownloadServer,
  SetNxmRedirectOtherGames,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { ExtensionContact } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { When } from '../../i18n/When.tsx'
import { openPage } from '../../mods/menu.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect, PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { prefCopy } from '../prefCopy.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useNxmHandler } from './nxmHandler.tsx'

// Nexus links carry no profile, so the target profile is per game: one row for each game Nexus serves.
function NxmDefaultProfile({ game, gameName }: { game: string; gameName: string }) {
  const { t, i18n } = useLingui()
  const push = useToasts((s) => s.push)
  const [profiles, setProfiles] = useState<Profile[]>([])
  const value = useSettings((s) => s.games?.[game]?.nxmDefaultProfile ?? '')
  useEffect(() => {
    ListProfiles(game)
      .then((ps) => setProfiles(ps ?? []))
      .catch(() => setProfiles([]))
  }, [game])
  const copy = prefCopy(i18n, 'nxmDefaultProfile')
  const options = [
    { value: '', label: t`Last opened profile` },
    ...profiles.filter((p) => !p.hidden).map((p) => ({ value: p.id, label: p.name })),
  ]
  return (
    <SettingRow label={t`${copy.label} (${gameName})`} description={copy.description}>
      <PrefSelect
        value={options.some((o) => o.value === value) ? value : ''}
        onChange={(v) =>
          persist(
            () => SetByKey('nxmDefaultProfile', v, game),
            push,
            t`Could not save that setting`,
          )
        }
        options={options}
        label={copy.label}
      />
    </SettingRow>
  )
}

function NxmDefaultProfiles() {
  const [games, setGames] = useState<GameInfo[]>([])
  useEffect(() => {
    ListGames()
      .then((gs) => setGames((gs ?? []).filter((g) => g.sources?.includes('nexus'))))
      .catch(() => setGames([]))
  }, [])
  return (
    <>
      {games.map((g) => (
        <NxmDefaultProfile key={g.id} game={g.id} gameName={g.name} />
      ))}
    </>
  )
}

const EXTENSION_RELEASE =
  'https://github.com/Rethunk-Tech/mortar-browser-extension/releases/latest/download'
const EXTENSION_ZIP = `${EXTENSION_RELEASE}/mortar-browser-extension.zip`
const EXTENSION_XPI = `${EXTENSION_RELEASE}/mortar-browser-extension.xpi`

// Until the browser stores list the extension, it is installed from its own repo's latest release.
function ExtensionConnection() {
  const { t } = useLingui()
  const [contact, setContact] = useState<{
    browser: string
    lastSeen: string
    mismatch?: string
  } | null>(null)
  useEffect(() => {
    ExtensionContact().then(setContact).catch(reportUnexpected)
  }, [])
  const connected = contact !== null && contact.browser !== ''
  const mismatchText: Partial<Record<string, string>> = {
    extensionTooOld: t`The browser extension is too old for this Mortar`,
    extensionTooNew: t`The browser extension is too new for this Mortar`,
  }
  const mismatch = mismatchText[contact?.mismatch ?? '']
  return (
    <SettingRow label={t`Connection`}>
      <Box sx={{ fontSize: 14 }}>
        {mismatch ??
          (connected ? (
            <>
              {t`Connected from ${contact.browser}, last seen `}
              <When value={contact.lastSeen} withTime={true} />
            </>
          ) : (
            t`Not connected yet`
          ))}
      </Box>
    </SettingRow>
  )
}

function ExtensionInstall() {
  const { t } = useLingui()
  return (
    <>
      <ExtensionConnection />
      <SettingRow
        label={t`Get the extension`}
        description={t`It marks Nexus Mods pages with what your profile already has and sends Mod Manager Download links to Mortar.`}
      >
        <Button variant="outlined" onClick={() => openPage(EXTENSION_ZIP)}>
          {t`Download the extension`}
        </Button>
      </SettingRow>
      <ExtensionSteps />
    </>
  )
}

function ExtensionSteps() {
  const { t } = useLingui()
  return (
    <Box
      component="ol"
      sx={{ m: 0, px: 2.5, pl: 5, pb: 1.5, fontSize: 14, color: 'text.secondary', lineHeight: 1.6 }}
    >
      <li>
        {t`Chrome, Edge or another Chromium browser: unzip the download, open chrome://extensions, turn on Developer mode and choose Load unpacked on the unzipped folder.`}
      </li>
      <li>
        {t`Firefox:`}{' '}
        <Link
          component="button"
          onClick={() => openPage(EXTENSION_XPI)}
          sx={{ font: 'inherit', verticalAlign: 'baseline' }}
        >
          {t`download the signed add-on`}
        </Link>{' '}
        {t`and open it in Firefox. If that release has none, unzip the download, open about:debugging, choose This Firefox, then Load Temporary Add-on and pick manifest.json; Firefox removes it when it restarts.`}
      </li>
    </Box>
  )
}

function NxmLinks() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const nxm = useNxmHandler()
  const nxmPrevious = useSettings((s) => s.nxmPrevious)
  const nxmPreviousName = useSettings((s) => s.nxmPreviousName)
  const redirectOther = useSettings((s) => s.nxmRedirectOtherGames ?? nxmPrevious !== '')
  const owner = nxm.handled ? 'Mortar' : nxm.owner
  const redirectName = nxmPreviousName || nxmPrevious
  return (
    <>
      <SettingRow
        label={t`Handle "Mod Manager Download" links`}
        description={`${t`Clicking these links on Nexus starts the download in Mortar.`} ${
          owner ? t`${owner} opens these links now.` : t`Mortar handles these links now.`
        }`}
      >
        <PrefSwitch
          checked={nxm.handled}
          onChange={(on) => nxm.toggle(on)}
          label={t`Handle "Mod Manager Download" links`}
        />
      </SettingRow>
      {nxm.handled && nxmPrevious ? (
        <SettingRow
          label={t`Send other games' links to ${redirectName}`}
          description={t`Links for games Mortar does not manage open in the app that had them before.`}
        >
          <PrefSwitch
            checked={redirectOther}
            onChange={(on) =>
              persist(() => SetNxmRedirectOtherGames(on), push, t`Could not save that setting`)
            }
            label={t`Send other games' links to ${redirectName}`}
          />
        </SettingRow>
      ) : null}
      {nxm.dialog}
    </>
  )
}

function PreferredServer() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const preferred = useSettings((s) => s.nexusPreferredDownloadServer)
  const seen = useSettings((s) => s.nexusSeenDownloadServers)
  if (!seen || seen.length === 0) {
    return null
  }
  const options = [
    { value: '', label: t`Automatic` },
    ...seen.map((server) => ({ value: server, label: server })),
  ]
  return (
    <SettingRow
      label={t`Preferred download server`}
      description={t`Premium downloads come from this Nexus server when it offers the file.`}
    >
      <PrefSelect
        value={options.some((o) => o.value === preferred) ? preferred : ''}
        onChange={(v) =>
          persist(() => SetNexusPreferredDownloadServer(v), push, t`Could not save that setting`)
        }
        options={options}
        label={t`Preferred download server`}
      />
    </SettingRow>
  )
}

export function Downloads() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const chooseFolder = () =>
    persist(
      async () => {
        const dir = await PickFolder(t`Download folder`)
        if (dir) {
          await SetByKey('downloadFolder', dir, '')
        }
      },
      push,
      t`Could not save that setting`,
    )
  return (
    <>
      <SettingsSection title={t`Nexus links`}>
        <NxmLinks />
        <NxmDefaultProfiles />
      </SettingsSection>
      <SettingsSection title={t`Browser extension`}>
        <ExtensionInstall />
        <PrefByKey prefKey="extensionConnection" />
      </SettingsSection>
      <SettingsSection title={t`Downloading`}>
        <PrefKeys
          keys={[
            'parallelDownloads',
            'autoRetryDownloads',
            'pauseDownloadsWhilePlaying',
            'verifyNexusMD5',
          ]}
        />
        <PreferredServer />
      </SettingsSection>
      <SettingsSection title={t`Files`}>
        <PrefByKey
          prefKey="downloadFolder"
          extra={
            <Button variant="outlined" onClick={chooseFolder}>
              {t`Change folder…`}
            </Button>
          }
        />
        <PrefByKey prefKey="keepDownloadArchives" />
      </SettingsSection>
    </>
  )
}
