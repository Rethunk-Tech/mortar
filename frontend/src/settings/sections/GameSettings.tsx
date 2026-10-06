import { useLingui } from '@lingui/react/macro'
import { Box, Button, FormControlLabel, Radio, RadioGroup } from '@mui/material'
import { System } from '@wailsio/runtime'
import { FolderOpen, Undo2 } from 'lucide-react'
import { useState } from 'react'
import type { Install } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import {
  ClearLaunchOption,
  LaunchOptions,
  LaunchOptionsStartLoader,
  ResetInstall,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { OpenSteamValidate } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import { PickFolder } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import {
  ChooseGameFolder,
  SetByKey,
  SetGameFolder,
  SetGameStore,
  SetTellWhenSmapiOut,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { LauncherLogo } from '../../brand/launchers/LauncherLogo.tsx'
import { gameInfo, useGameInfo, useGameName } from '../../games/info.ts'
import { storeName } from '../../games/storeName.ts'
import { currentGame, useCurrentGame } from '../../nav/currentGame.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { FlatpakGrant } from '../../shell/FlatpakGrant.tsx'
import { type InlineError, inlineError, reportError, toastError } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSwitch } from '../PrefControls.tsx'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { Searchable, SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'
import { BackupsUsageRow } from './DataBackups.tsx'
import { LoaderRow } from './LoaderRow.tsx'
import { ScheduledStatus } from './ScheduledStatus.tsx'

// Without a loader's reading, an install's version is Steam's build id: digits only, never a dotted version.
const STEAM_BUILD = /^\d+$/

const noShrink = { flexShrink: 0 }

// A GOG install found through Heroic or Minigalaxy carries that launcher's logo.
const GOG_VIA = /^gog-/
const launcherOf = (store: string) => store.replace(GOG_VIA, '')

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
  const loader = useGameInfo()?.loader ?? ''
  return (
    <ConfirmDialog
      open={open}
      title={t`Reset game install?`}
      body={t`This deletes the game folder at ${folder}, with every file in it, including ${loader} and mods placed there. Your saves and your profiles' mods are stored elsewhere and are kept.`}
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
    const options = await LaunchOptions(currentGame())
    if (await LaunchOptionsStartLoader(currentGame(), options)) {
      push({
        kind: 'success',
        title,
        action: {
          label,
          run: () => ClearLaunchOption(currentGame()).then(undefined, reportError(errorTitle)),
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
  installs: Install[]
  store: string
  override: string
  onPick: (store: string) => void
}) {
  const { t } = useLingui()
  if (installs.length <= 1) {
    return null
  }
  return (
    <RadioGroup
      aria-label={t`Game install`}
      value={override ? '' : store}
      onChange={(e) => onPick(e.target.value)}
    >
      {installs.map((item) => (
        <FormControlLabel
          key={`${item.store}:${item.dir}`}
          value={item.store}
          control={<Radio size="small" />}
          label={
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, minWidth: 0 }}>
              <LauncherLogo id={launcherOf(item.store)} size={22} />
              <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
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
  installs: Install[]
  onRefresh: () => void
}) {
  const { t } = useLingui()
  const game = useCurrentGame()
  const gameName = useGameName(game)
  const override = useSettings((s) => s.gameFolders?.[game] ?? '')
  const [error, setError] = useState<InlineError | null>(null)
  const [resetting, setResetting] = useState(false)
  const [restoreBusy, setRestoreBusy] = useState(false)
  const change = (run: Promise<void>) => {
    setError(null)
    run.then(onRefresh).catch((e: unknown) => setError(inlineError(e)))
  }
  const runtimeVersion = installs.find((i) => i.dir === folder)?.runtimeVersion
  const gameVersion = installs.find((i) => i.dir === folder)?.version
  const known = storeName(store)
  const foundIn = known ? t(known) : t`Steam`
  let source = t`${gameName} was not found. Browse to its folder.`
  if (override) {
    source = t`Chosen by you`
  } else if (folder) {
    source = t`Found in ${{ path: foundIn }}`
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
              onClick={() => change(ChooseGameFolder(game))}
              sx={noShrink}
            >
              {t`Change folder…`}
            </Button>
            {override ? (
              <Button
                variant="outlined"
                startIcon={<Undo2 size={16} />}
                onClick={() => change(SetGameFolder(game, ''))}
                sx={noShrink}
              >
                {t`Use default`}
              </Button>
            ) : null}
          </Box>
        </SettingRow>
        {gameVersion ? (
          <SettingRow
            label={STEAM_BUILD.test(gameVersion) ? t`Steam build` : t`Game version`}
            description={t`Read from the game's install.`}
          >
            <Box sx={{ fontSize: 14, color: 'var(--mortar-ink-sec)' }}>{gameVersion}</Box>
          </SettingRow>
        ) : null}
        {runtimeVersion ? (
          <SettingRow
            label={t`Compatibility tool`}
            description={t`Chosen in the game's Steam properties.`}
          >
            <Box sx={{ fontSize: 14, color: 'var(--mortar-ink-sec)' }}>{runtimeVersion}</Box>
          </SettingRow>
        ) : null}
        {error ? (
          <Box
            role="alert"
            title={error.details}
            sx={{ px: 2.5, py: 1.5, fontSize: 14, color: 'error.light' }}
          >
            {error.message}
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
                  change(SetGameFolder(game, '').then(() => SetGameStore(game, next)))
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
          description={t`Delete the game folder, then reinstall ${gameName} from your launcher.`}
        >
          <Button
            variant="outlined"
            color="error"
            disabled={resetting || !folder}
            onClick={() => setResetting(true)}
            sx={noShrink}
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
          setError(null)
          setRestoreBusy(true)
          try {
            await ResetInstall(game)
            if (System.IsWindows()) {
              await offerLaunchOptionRemoval(
                useToasts.getState().push,
                t`Game install deleted`,
                t`Remove SMAPI from Steam's launch options`,
                t`Could not update Steam's launch options`,
              )
            }
            if (store === 'steam' || store === 'flatpak-steam') {
              await OpenSteamValidate(gameInfo(game)?.appId ?? '')
            } else {
              useToasts.getState().push({
                kind: 'info',
                title: t`Game install deleted`,
                body: t`Reinstall ${gameName} from your game launcher.`,
              })
            }
            onRefresh()
            setResetting(false)
          } catch (e: unknown) {
            toastError(t`Could not reset the game install`, e)
          } finally {
            setRestoreBusy(false)
          }
        }}
      />
    </>
  )
}

function LoaderPage({
  loader,
  onVersion,
}: {
  loader: { id: string; name: string }
  onVersion: (v: string) => void
}) {
  const { t } = useLingui()
  const game = useCurrentGame()
  const tellWhenSmapiOut = useSettings((s) => s.tellWhenSmapiOut) !== false
  const push = useToasts((s) => s.push)
  return (
    <SettingsSection title={loader.name}>
      <LoaderRow loader={loader} onVersion={onVersion} />
      <SettingRow label={t`Tell me when a new mod loader version is out`}>
        <PrefSwitch
          checked={tellWhenSmapiOut}
          onChange={(on) =>
            persist(() => SetTellWhenSmapiOut(on), push, t`Could not save that setting`)
          }
          label={t`Tell me when a new mod loader version is out`}
        />
      </SettingRow>
      {loader.id === 'smapi' ? <PrefByKey prefKey="smapiBuilds" game={game} /> : null}
    </SettingsSection>
  )
}

function ExtraModsFolder() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const push = useToasts((s) => s.push)
  const folder = useSettings((s) => s.games?.[game]?.extraModsFolder ?? '')
  const choose = () =>
    persist(
      async () => {
        const dir = await PickFolder(t`Extra mods folder`)
        if (dir) {
          await SetByKey('extraModsFolder', dir, game)
        }
      },
      push,
      t`Could not save that setting`,
    )
  const clear = () =>
    persist(() => SetByKey('extraModsFolder', '', game), push, t`Could not save that setting`)
  return (
    <PrefByKey
      prefKey="extraModsFolder"
      game={game}
      extra={
        <Box sx={{ display: 'flex', gap: 1, flexShrink: 0 }}>
          {folder ? (
            <Button onClick={clear} sx={noShrink}>
              {t`Clear`}
            </Button>
          ) : null}
          <Button variant="outlined" onClick={choose} sx={noShrink}>
            {t`Choose folder…`}
          </Button>
        </Box>
      }
    />
  )
}

function BackupsPage() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const scheduleOff = useSettings((s) => (s.games?.[game]?.saveBackupHours ?? 0) === 0)
  const push = useToasts((s) => s.push)
  const chooseBackupLocation = () =>
    persist(
      async () => {
        const dir = await PickFolder(t`Backup location`)
        if (dir) {
          await SetByKey('backupLocation', dir, game)
        }
      },
      push,
      t`Could not save that setting`,
    )
  return (
    <SettingsSection title={t`Save backups`}>
      <PrefKeys keys={['backupBeforePlay', 'saveBackupsKept', 'saveBackupHours']} game={game} />
      <ScheduledStatus />
      <PrefByKey
        prefKey="saveBackupKeep"
        game={game}
        disabledReason={scheduleOff ? t`Enable scheduled save backups first.` : ''}
      />
      <PrefByKey
        prefKey="backupLocation"
        game={game}
        extra={
          <Button variant="outlined" onClick={chooseBackupLocation} sx={noShrink}>
            {t`Change folder…`}
          </Button>
        }
      />
      <BackupsUsageRow />
    </SettingsSection>
  )
}

export { BackupsPage, ExtraModsFolder, GameFolder, LoaderPage }
