import { useLingui } from '@lingui/react/macro'
import { Button, Tooltip } from '@mui/material'
import { useEffect, useState } from 'react'
import { List as ListGames } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import { PickFolder } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import {
  SetByKey,
  SetNexusPreferredDownloadServer,
  SetNxmDefaultProfile,
  SetNxmRedirectOtherGames,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect, PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { prefCopy } from '../prefCopy.ts'
import { GAME_STARDEW } from '../prefValue.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { useNxmHandler } from './nxmHandler.tsx'
import { nxmOwnerName } from './nxmOwnerName.ts'

// Nexus links carry no profile, so the target profile is per game; Stardew is the only game today.
function NxmDefaultProfile() {
  const { t, i18n } = useLingui()
  const push = useToasts((s) => s.push)
  const profiles = useProfiles((s) => s.profiles)
  const value = useSettings((s) => s.games?.[GAME_STARDEW]?.nxmDefaultProfile ?? '')
  const [gameName, setGameName] = useState('')
  useEffect(() => {
    ListGames()
      .then((gs) => setGameName((gs ?? []).find((g) => g.id === GAME_STARDEW)?.name ?? ''))
      .catch(() => setGameName(''))
  }, [])
  const copy = prefCopy(i18n, 'nxmDefaultProfile')
  const options = [
    { value: '', label: t`Last opened profile` },
    ...(profiles ?? []).filter((p) => !p.hidden).map((p) => ({ value: p.id, label: p.name })),
  ]
  return (
    <SettingRow
      label={gameName ? t`${copy.label} (${gameName})` : copy.label}
      description={copy.description}
    >
      <PrefSelect
        value={options.some((o) => o.value === value) ? value : ''}
        onChange={(v) =>
          persist(() => SetNxmDefaultProfile(v), push, t`Couldn't save that setting`)
        }
        options={options}
        label={copy.label}
      />
    </SettingRow>
  )
}

function NxmLinks() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const nxm = useNxmHandler()
  const nxmPrevious = useSettings((s) => s.nxmPrevious)
  const nxmPreviousName = useSettings((s) => s.nxmPreviousName)
  const redirectOther = useSettings((s) => s.nxmRedirectOtherGames ?? nxmPrevious !== '')
  const owner = nxmOwnerName(nxm.handled, nxm.owner)
  const redirectName = nxmOwnerName(true, nxmPreviousName || nxmPrevious)
  return (
    <>
      <SettingRow
        label={t`Handle "Mod Manager Download" links`}
        description={t`Clicking these links on Nexus starts the download in Mortar.`}
      >
        <Tooltip
          title={owner ? t`${owner} opens these links now.` : t`Mortar handles these links now.`}
        >
          <PrefSwitch
            checked={nxm.handled}
            onChange={(on) => nxm.toggle(on)}
            label={t`Handle "Mod Manager Download" links`}
          />
        </Tooltip>
      </SettingRow>
      {nxm.handled && nxmPrevious ? (
        <SettingRow
          label={t`Send other games' links to ${redirectName}`}
          description={t`Links for games Mortar does not manage open in the app that had them before.`}
        >
          <PrefSwitch
            checked={redirectOther}
            onChange={(on) =>
              persist(() => SetNxmRedirectOtherGames(on), push, t`Couldn't save that setting`)
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
          persist(() => SetNexusPreferredDownloadServer(v), push, t`Couldn't save that setting`)
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
      t`Couldn't save that setting`,
    )
  return (
    <>
      <SettingsSection title={t`Nexus links`}>
        <NxmLinks />
        <PrefByKey prefKey="extensionConnection" />
        <NxmDefaultProfile />
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
            <Button variant="outlined" onClick={chooseFolder} sx={{ whiteSpace: 'nowrap' }}>
              {t`Change folder…`}
            </Button>
          }
        />
        <PrefByKey prefKey="keepDownloadArchives" />
      </SettingsSection>
    </>
  )
}
