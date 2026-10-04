import { useLingui } from '@lingui/react/macro'
import { Box, Button, FormControlLabel, Radio, RadioGroup } from '@mui/material'
import { Browser, System } from '@wailsio/runtime'
import { Download, FolderOpen, Undo2 } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type { FoundInstall } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import {
  ClearLaunchOption,
  LaunchOptions,
  ResetInstall,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import { State } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { PickFolder } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import {
  ChooseGameFolder,
  SetByKey,
  SetGameFolder,
  SetGameStore,
  SetTellWhenSmapiOut,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { launchOptionsSet } from '../../firstrun/logic.ts'
import { storeName } from '../../games/storeName.ts'
import { useLaunch } from '../../launch/store.ts'
import { InstallSteps } from '../../loader/InstallSteps.tsx'
import { useLoader } from '../../loader/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { FlatpakGrant } from '../../shell/FlatpakGrant.tsx'
import { errorText, reportError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { Searchable, SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { SmapiVersionRow } from './SmapiVersionRow.tsx'

const GAME = 'stardew'

const outline = { whiteSpace: 'nowrap', flexShrink: 0, height: 42 }
const nowrap = { whiteSpace: 'nowrap', flexShrink: 0 }

function StoreLabel({ store }: { store: string }) {
  const { t } = useLingui()
  const named = storeName(store)
  return named ? t(named) : t`Steam`
}

function ResetInstallDialog({
  open,
  folder,
  busy,
  onClose,
  onConfirm,
}: {
  open: boolean
  folder: string
  busy: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  const { t } = useLingui()
  return (
    <ConfirmDialog
      open={open}
      title={t`Reset game install?`}
      body={t`This deletes the game folder at ${folder}, including every file in it, SMAPI, and any mods placed there. Saves are not in this folder and will be kept. Profiles' mods are stored separately by Mortar and will be kept.`}
      confirmLabel={t`Reset install`}
      color="error"
      busy={busy}
      onCancel={onClose}
      onConfirm={onConfirm}
    />
  )
}

async function offerLaunchOptionRemoval(
  push: ReturnType<typeof useToasts.getState>['push'],
  title: string,
  label: string,
  errorTitle: string,
) {
  try {
    const options = await LaunchOptions(GAME)
    if (launchOptionsSet(options)) {
      push({
        kind: 'success',
        title,
        action: {
          label,
          run: () => ClearLaunchOption(GAME).then(undefined, reportError(errorTitle)),
        },
      })
    }
  } catch {
    // Steam may not be installed; reset still succeeded.
  }
}

function ExtraInstalls({
  installs,
  store,
  override,
  onPick,
}: {
  installs: FoundInstall[]
  store: string
  override: string
  onPick: (store: string) => void
}) {
  if (installs.length <= 1) {
    return null
  }
  return (
    <RadioGroup value={override ? '' : store} onChange={(e) => onPick(e.target.value)}>
      {installs.map((item) => (
        <FormControlLabel
          key={`${item.store}:${item.dir}`}
          value={item.store}
          control={<Radio size="small" />}
          label={
            <Box sx={{ display: 'flex', flexDirection: 'column' }}>
              <StoreLabel store={item.store} />
              <Box
                sx={{
                  fontSize: 12,
                  color: 'var(--mortar-ink-sec)',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
                title={item.dir}
              >
                {item.dir}
              </Box>
            </Box>
          }
        />
      ))}
    </RadioGroup>
  )
}

function GameFolder({
  folder,
  versionNote,
  store,
  installs,
  onRefresh,
}: {
  folder: string
  versionNote: string
  store: string
  installs: FoundInstall[]
  onRefresh: () => void
}) {
  const { t } = useLingui()
  const override = useSettings((s) => s.gameFolders?.[GAME] ?? '')
  const [error, setError] = useState('')
  const [resetting, setResetting] = useState(false)
  const [restoreBusy, setRestoreBusy] = useState(false)
  const change = (run: Promise<void>) => {
    setError('')
    run
      .then(onRefresh)
      .catch((e: unknown) => setError(errorText(e) ?? t`That folder cannot be used`))
  }
  const known = storeName(store)
  const foundIn = known ? t(known) : t`Steam`
  let source = t`Stardew Valley was not found. Browse to its folder.`
  if (override) {
    source = t`Chosen by you`
  } else if (folder) {
    source = t`Found in ${foundIn}`
  }
  return (
    <>
      <SettingsSection title={t`Game folder`}>
        <SettingRow
          label={t`Game folder`}
          description={`${folder || t`Not found`} · ${source}${versionNote}`}
        >
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="outlined"
              startIcon={<FolderOpen size={16} />}
              onClick={() => change(ChooseGameFolder(GAME))}
              sx={nowrap}
            >
              {t`Change folder…`}
            </Button>
            {override ? (
              <Button
                variant="outlined"
                startIcon={<Undo2 size={16} />}
                onClick={() => change(SetGameFolder(GAME, ''))}
                sx={nowrap}
              >
                {t`Use default`}
              </Button>
            ) : null}
          </Box>
        </SettingRow>
        {error ? (
          <Box role="alert" sx={{ px: 2.5, py: 1.5, fontSize: 14, color: 'error.light' }}>
            {error}
          </Box>
        ) : null}
        {installs.length > 1 ? (
          <Searchable
            terms={`${t`Game folder`} ${installs.map((i) => `${i.store} ${i.dir}`).join(' ')}`}
          >
            <Box sx={{ px: 2.5, py: 1.5 }}>
              <ExtraInstalls
                installs={installs}
                store={store}
                override={override}
                onPick={(next) =>
                  change(SetGameFolder(GAME, '').then(() => SetGameStore(GAME, next)))
                }
              />
            </Box>
          </Searchable>
        ) : null}
        <Searchable terms={`Flatpak Steam ${t`Grant access`}`}>
          <FlatpakGrant />
        </Searchable>
      </SettingsSection>
      <SettingsSection title={t`Reset`}>
        <SettingRow
          label={t`Reset game install`}
          description={t`Delete the game folder, then reinstall Stardew Valley from your launcher.`}
        >
          <Button
            variant="outlined"
            color="error"
            disabled={resetting || !folder}
            onClick={() => setResetting(true)}
            sx={nowrap}
          >
            {t`Reset…`}
          </Button>
        </SettingRow>
      </SettingsSection>
      <ResetInstallDialog
        open={resetting}
        folder={folder}
        busy={restoreBusy}
        onClose={() => {
          if (!restoreBusy) {
            setResetting(false)
          }
        }}
        onConfirm={async () => {
          setError('')
          setRestoreBusy(true)
          try {
            await ResetInstall(GAME)
            if (System.IsWindows()) {
              await offerLaunchOptionRemoval(
                useToasts.getState().push,
                t`Game install deleted`,
                t`Remove SMAPI from Steam's launch options`,
                t`Couldn't update Steam's launch options`,
              )
            }
            if (store === 'steam' || store === 'flatpak-steam') {
              await Browser.OpenURL('steam://validate/413150')
            } else {
              useToasts.getState().push({
                kind: 'info',
                title: t`Game install deleted`,
                body: t`Reinstall Stardew Valley from your game launcher.`,
              })
            }
            onRefresh()
            setResetting(false)
          } catch (e: unknown) {
            useToasts.getState().push({
              kind: 'error',
              title: errorText(e) ?? t`Could not reset the game install`,
            })
          } finally {
            setRestoreBusy(false)
          }
        }}
      />
    </>
  )
}

function Smapi({ onVersion }: { onVersion: (v: string) => void }) {
  const { t } = useLingui()
  const status = useLoader((s) => s.status)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const check = useLoader((s) => s.check)
  const install = useLoader((s) => s.install)
  const pending = useLoader((s) => s.pending)
  const refreshLaunch = useLaunch((s) => s.refresh)
  const playing = useLaunch(
    (s) =>
      s.starting ||
      (s.status?.game === GAME &&
        (s.status.state === State.Launching || s.status.state === State.Running)),
  )
  useEffect(() => {
    check(GAME)
    refreshLaunch(GAME)
  }, [check, refreshLaunch])
  const gameVersion = status?.gameVersion ?? ''
  useEffect(() => {
    onVersion(gameVersion)
  }, [gameVersion, onVersion])
  const locked = pending || playing
  const lockReason = playing
    ? t`Stop the game to install SMAPI.`
    : t`An install is already running.`
  let title = t`SMAPI is not installed`
  let detail = ''
  let action = t`Install`
  if (status?.broken) {
    title = t`A game update replaced SMAPI's launcher`
    action = t`Reinstall`
  } else if (status?.installed) {
    title = t`SMAPI ${status.version}`
    action = status.updateAvailable ? t`Update` : t`Reinstall`
    detail = status.updateAvailable
      ? t`SMAPI ${status.latest} is available`
      : t`Installed and up to date`
  }
  let control: ReactNode = null
  if (installing) {
    control = <InstallSteps steps={steps} />
  } else if (status) {
    control = (
      <DisabledReason title={lockReason} disabled={locked}>
        <Button
          variant="outlined"
          startIcon={<Download size={16} />}
          disabled={locked}
          onClick={() => install(GAME)}
          sx={{ ...outline, height: 38 }}
        >
          {action}
        </Button>
      </DisabledReason>
    )
  }
  return (
    <SettingRow label={title} description={detail} block={installing}>
      {control}
    </SettingRow>
  )
}

function SmapiPage({ onVersion }: { onVersion: (v: string) => void }) {
  const { t } = useLingui()
  const tellWhenSmapiOut = useSettings((s) => s.tellWhenSmapiOut) !== false
  const push = useToasts((s) => s.push)
  return (
    <SettingsSection title={t`SMAPI`}>
      <Smapi onVersion={onVersion} />
      <SettingRow label={t`Tell me when a new SMAPI is out`}>
        <PrefSwitch
          checked={tellWhenSmapiOut}
          onChange={(on) =>
            persist(() => SetTellWhenSmapiOut(on), push, t`Couldn't save that setting`)
          }
          label={t`Tell me when a new SMAPI is out`}
        />
      </SettingRow>
      <SmapiVersionRow />
      <PrefByKey prefKey="smapiBuilds" game={GAME} />
    </SettingsSection>
  )
}

function BackupsPage() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const chooseBackupLocation = () =>
    persist(
      async () => {
        const dir = await PickFolder(t`Backup location`)
        if (dir) {
          await SetByKey('backupLocation', dir, GAME)
        }
      },
      push,
      t`Couldn't save that setting`,
    )
  return (
    <SettingsSection title={t`Play backups`}>
      <PrefKeys keys={['backupBeforePlay', 'launchBackupsKept']} game={GAME} />
      <PrefByKey
        prefKey="backupLocation"
        game={GAME}
        extra={
          <Button variant="outlined" onClick={chooseBackupLocation} sx={nowrap}>
            {t`Change folder…`}
          </Button>
        }
      />
    </SettingsSection>
  )
}

export { BackupsPage, GameFolder, SmapiPage }
