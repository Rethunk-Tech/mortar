import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { memo, useCallback, useEffect, useState } from 'react'
import type { LinkSource } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/models.ts'
import {
  DisableSource,
  EnableSource,
  LinkSources,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/service.ts'
import type { Profile } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  SetByKey,
  SetNexusPreferredDownloadServer,
  SetNxmRedirectOtherGames,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { ExtensionContact } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/support/service.ts'
import { CHROME, CHROMIUM, EDGE, FIREFOX } from '../../brand/productNames.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { useGames } from '../../games/info.ts'
import { When } from '../../i18n/When.tsx'
import { openPage } from '../../mods/menu.ts'
import { listProfiles } from '../../profiles/listProfiles.ts'
import { useProfiles } from '../../profiles/store.ts'
import { space } from '../../theme/density.ts'
import { reportUnexpected, toastError } from '../../toasts/report.ts'
import { PrefSelect, PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { pickFolderSetting, saveSetting } from '../persist.ts'
import { prefCopy } from '../prefCopy.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useSettingsRelevant } from '../useSettingsSearch.ts'
import { useNxmHandler } from './nxmHandler.tsx'
import { WatchFolders } from './WatchFolders.tsx'

// Nexus links carry no profile, so the target profile is per game: one row for each game Nexus serves.
interface Named {
  id: string
  name: string
}

const namesOf = (profiles: Profile[]): Named[] => profiles.map((p) => ({ id: p.id, name: p.name }))

function NxmDefaultProfile({ game, gameName }: { game: string; gameName: string }) {
  const { t, i18n } = useLingui()
  // Names only, started from the open game's profiles when this is that game, so the read below changes nothing on
  // screen unless a profile was added, removed or renamed.
  const value = useSettings((s) => s.games?.[game]?.nxmDefaultProfile ?? '')
  const copy = prefCopy(i18n, 'nxmDefaultProfile')
  const label = t`${{ name: copy.label }} (${{ game: gameName }})`
  const relevant = useSettingsRelevant(label, copy.description ?? '')
  const open = useProfiles.getState()
  const { data: profiles } = useLoaded<Named[]>(
    relevant
      ? () =>
          listProfiles(game).then(
            (ps) => namesOf(ps ?? []),
            () => [],
          )
      : null,
    [game, relevant],
    open.game?.id === game ? namesOf(open.profiles) : [],
  )
  const options = [
    { value: '', label: t`Last opened profile` },
    ...profiles.map((p) => ({ value: p.id, label: p.name })),
  ]
  return (
    <SettingRow label={label} description={copy.description}>
      <PrefSelect
        value={options.some((o) => o.value === value) ? value : ''}
        onChange={(v) => saveSetting(() => SetByKey('nxmDefaultProfile', v, game))}
        options={options}
        label={copy.label}
      />
    </SettingRow>
  )
}

function NxmDefaultProfiles() {
  const games = useGames().filter((g) => g.installed && g.sources?.includes('nexus'))
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

// The extension may connect while this page is open.
const EXTENSION_POLL_MS = 5000

// Until the browser stores list the extension, it is installed from its own repo's latest release.
function ExtensionConnection() {
  const { t } = useLingui()
  const [contact, setContact] = useState<{
    browser: string
    lastSeen: string
    mismatch?: string
  } | null>(null)
  const relevant = useSettingsRelevant(t`Connection`)
  useEffect(() => {
    if (!relevant) {
      return
    }
    const read = () => ExtensionContact().then(setContact).catch(reportUnexpected)
    read()
    const timer = setInterval(read, EXTENSION_POLL_MS)
    return () => clearInterval(timer)
  }, [relevant])
  const connected = contact !== null && contact.browser !== ''
  const mismatchText: Partial<Record<string, string>> = {
    extensionTooOld: t`The browser extension is too old for this Mortar`,
    extensionTooNew: t`The browser extension is too new for this Mortar`,
  }
  const mismatch = mismatchText[contact?.mismatch ?? '']
  const status = connected ? (
    <>
      {t`Connected from ${{ browser: contact.browser }}, last seen `}
      <When value={contact.lastSeen} withTime={true} />
    </>
  ) : (
    t`Not connected yet`
  )
  return (
    <SettingRow label={t`Connection`}>
      <Box sx={{ fontSize: 'medium.fontSize' }}>
        {contact === null ? null : (mismatch ?? status)}
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
        description={
          <>
            {t`It marks ${{ source: sourceLabel('nexus') }} pages with what your profile already has and sends Mod Manager Download links to Mortar.`}
            <ExtensionSteps />
          </>
        }
      >
        <Button variant="outlined" onClick={() => openPage(EXTENSION_ZIP)}>
          {t`Download the extension`}
        </Button>
      </SettingRow>
    </>
  )
}

function ExtensionSteps() {
  const { t } = useLingui()
  return (
    <Box
      component="ol"
      // Inside the row's description, so it takes the description's type and only needs the list's indent.
      sx={{ m: 0, mt: space.gap, pl: 2.5 }}
    >
      <li>
        {t`${{ app: CHROME }}, ${{ otherApp: EDGE }} or another ${{ thirdApp: CHROMIUM }} browser: unzip the download, open chrome://extensions, enable Developer mode and choose Load unpacked on the unzipped folder.`}
      </li>
      <li>
        {t`${{ app: FIREFOX }}:`}{' '}
        <Link
          component="button"
          onClick={() => openPage(EXTENSION_XPI)}
          sx={{ font: 'inherit', verticalAlign: 'baseline' }}
        >
          {t`download the signed add-on`}
        </Link>{' '}
        {t`and open it in ${{ app: FIREFOX }}. If that release has none, unzip the download, open about:debugging, choose This ${{ app: FIREFOX }}, then Load Temporary Add-on and pick manifest.json; ${{ app: FIREFOX }} removes it when it restarts.`}
      </li>
    </Box>
  )
}

// One row for each source with a link scheme that Mortar handles only on request (Thunderstore's ror2mm).
// Memoised: the Nexus link status arriving re-renders the rows above, not every source's link row.
const SourceLinks = memo(function SourceLinksView() {
  const { t } = useLingui()
  const [sources, setSources] = useState<LinkSource[]>([])
  const load = useCallback(() => {
    LinkSources()
      .then((list) => setSources(list ?? []))
      .catch(reportUnexpected)
  }, [])
  useEffect(load, [load])
  return sources.map((source) => (
    <SettingRow
      key={source.id}
      label={t`Handle ${{ name: source.name }} links`}
      description={t`Opening a ${{ scheme: source.scheme }}:// link, such as Install with Mod Manager, installs the mod in Mortar.`}
    >
      <PrefSwitch
        checked={source.handled}
        onChange={(on) => {
          const done = on ? EnableSource(source.id) : DisableSource(source.id)
          done
            .catch((error: unknown) => toastError(t`Could not change how mod links open`, error))
            .finally(load)
        }}
        label={t`Handle ${{ name: source.name }} links`}
      />
    </SettingRow>
  ))
})

function NxmLinks() {
  const { t } = useLingui()
  const nxm = useNxmHandler()
  const nxmPrevious = useSettings((s) => s.nxmPreviousHandlers?.nxm ?? '')
  const nxmPreviousName = useSettings((s) => s.nxmPreviousName)
  const redirectOther = useSettings((s) => s.nxmRedirectOtherGames ?? nxmPrevious !== '')
  const owner = nxm.handled ? 'Mortar' : nxm.owner
  const redirectName = nxmPreviousName || nxmPrevious
  return (
    <>
      <SettingRow
        label={t`Handle "Mod Manager Download" links`}
        description={`${t`Clicking these links on ${{ source: sourceLabel('nexus') }} starts the download in Mortar.`} ${
          owner ? t`${owner} opens these links now.` : t`No app opens these links now.`
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
          label={t`Send other games' links to ${{ name: redirectName }}`}
          description={t`Links for games Mortar does not manage open in the app that had them before.`}
        >
          <PrefSwitch
            checked={redirectOther}
            onChange={(on) => saveSetting(() => SetNxmRedirectOtherGames(on))}
            label={t`Send other games' links to ${{ name: redirectName }}`}
          />
        </SettingRow>
      ) : null}
      <SourceLinks />
      {nxm.dialog}
    </>
  )
}

function PreferredServer() {
  const { t } = useLingui()
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
      description={t`Premium downloads come from this ${{ source: sourceLabel('nexus') }} server when it offers the file.`}
    >
      <PrefSelect
        value={options.some((o) => o.value === preferred) ? preferred : ''}
        onChange={(v) => saveSetting(() => SetNexusPreferredDownloadServer(v))}
        options={options}
        label={t`Preferred download server`}
      />
    </SettingRow>
  )
}

export function Downloads() {
  const { t } = useLingui()
  const chooseFolder = () =>
    pickFolderSetting(t`Download folder`, 'downloadFolder', '', t`Could not save that setting`)
  return (
    <>
      <SettingsSection title={t`Links`}>
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
            'downloadSpeedLimit',
            'autoRetryDownloads',
            'thunderstoreHost',
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
        <WatchFolders />
      </SettingsSection>
    </>
  )
}
