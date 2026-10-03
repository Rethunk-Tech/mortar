import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControlLabel,
  Radio,
  RadioGroup,
  Switch,
} from '@mui/material'
import { Browser, Clipboard, System } from '@wailsio/runtime'
import { Copy, Download, FolderOpen, Undo2 } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type { FoundInstall } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import {
  ClearLaunchOption,
  GrantSteamAccess,
  LaunchOptions,
  ResetInstall,
  SteamAccess,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import { State } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import {
  ChooseGameFolder,
  SetGameFolder,
  SetGameStore,
  SetTellWhenSmapiOut,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { launchOptionsSet } from '../../firstrun/logic.ts'
import { loadGameStatus } from '../../games/status.ts'
import { storeName } from '../../games/storeName.ts'
import { useLaunch } from '../../launch/store.ts'
import { InstallSteps } from '../../loader/InstallSteps.tsx'
import { useLoader } from '../../loader/store.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { useSettings } from '../store.ts'
import { StreamOverlay } from './StreamOverlay.tsx'

const GAME = 'stardew'

const outline = { whiteSpace: 'nowrap', flexShrink: 0, height: 42 }

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
    <Dialog open={open} onClose={busy ? undefined : onClose} transitionDuration={0}>
      <DialogTitle>{t`Reset game install?`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`This deletes the game folder at ${folder}, including every file in it, SMAPI, and any mods placed there. Saves are not in this folder and will be kept. Profiles' mods are stored separately by Mortar and will be kept.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {t`Cancel`}
        </Button>
        <Button color="error" variant="contained" onClick={onConfirm} disabled={busy}>
          {t`Delete and restore`}
        </Button>
      </DialogActions>
    </Dialog>
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
          run: () =>
            ClearLaunchOption(GAME).then(undefined, (err: unknown) => {
              const body = errorText(err)
              push({ kind: 'error', title: errorTitle, ...(body ? { body } : {}) })
            }),
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
                  color: 'rgba(225,225,230,0.95)',
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
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Game folder`}</Box>
      <ExtraInstalls
        installs={installs}
        store={store}
        override={override}
        onPick={(next) => change(SetGameFolder(GAME, '').then(() => SetGameStore(GAME, next)))}
      />
      <Box sx={{ display: 'flex', gap: 1 }}>
        <Box
          role="textbox"
          aria-readonly={true}
          aria-label={t`Game folder`}
          title={folder || undefined}
          sx={{
            flexGrow: 1,
            minWidth: 0,
            height: 42,
            px: '12px',
            display: 'flex',
            alignItems: 'center',
            bgcolor: 'rgba(0,0,0,0.4)',
            border: '1px solid rgba(255,255,255,0.18)',
            borderRadius: '6px',
            fontSize: 13,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {folder || t`Not found`}
        </Box>
        <Button
          variant="outlined"
          startIcon={<FolderOpen size={16} />}
          onClick={() => change(ChooseGameFolder(GAME))}
          sx={outline}
        >
          {t`Browse…`}
        </Button>
        {override ? (
          <Button
            variant="outlined"
            startIcon={<Undo2 size={16} />}
            onClick={() => change(SetGameFolder(GAME, ''))}
            sx={outline}
          >
            {t`Use discovered`}
          </Button>
        ) : null}
      </Box>
      <Button
        variant="text"
        color="error"
        disabled={resetting || !folder}
        onClick={() => setResetting(true)}
        sx={{ alignSelf: 'flex-start', minHeight: 36 }}
      >
        {t`Reset game install`}
      </Button>
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
            setError(errorText(e) ?? t`That folder cannot be used`)
          } finally {
            setRestoreBusy(false)
          }
        }}
      />
      {error ? (
        <Box role="alert" sx={{ fontSize: 13, color: 'error.light' }}>
          {error}
        </Box>
      ) : (
        <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
          {source}
          {versionNote}
        </Box>
      )}
    </Box>
  )
}

function FlatpakAccess() {
  const { t } = useLingui()
  const [cmd, setCmd] = useState('')
  const [needed, setNeeded] = useState(false)
  const [granted, setGranted] = useState(false)
  const [ask, setAsk] = useState(false)
  const load = () => {
    SteamAccess()
      .then((a) => {
        setCmd(a.command)
        setNeeded(a.needed)
        setGranted(a.granted)
      })
      .catch(reportUnexpected)
  }
  useEffect(load, [])
  if (!needed || granted) {
    return null
  }
  return (
    <>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Flatpak Steam cannot read your mods`}</Box>
        <Box sx={{ fontSize: 13, color: 'rgba(225,225,230,0.95)' }}>
          {t`Grant the Steam sandbox read access to Mortar's data folder, or SMAPI will not see this profile's mods.`}
        </Box>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start' }}>
          <Box
            sx={{
              flex: 1,
              minWidth: 0,
              px: 1.5,
              py: 0.75,
              bgcolor: 'rgba(0,0,0,0.45)',
              border: '1px solid rgba(255,255,255,0.15)',
              borderRadius: '6px',
              fontFamily: 'monospace',
              fontSize: 13,
              wordBreak: 'break-all',
              userSelect: 'text',
            }}
          >
            {cmd}
          </Box>
          <Button
            variant="outlined"
            startIcon={<Copy size={16} />}
            onClick={() => {
              Clipboard.SetText(cmd).then(
                () => useToasts.getState().push({ kind: 'success', title: t`Command copied` }),
                reportUnexpected,
              )
            }}
            sx={outline}
          >
            {t`Copy`}
          </Button>
          <Button variant="contained" onClick={() => setAsk(true)} sx={outline}>
            {t`Grant access`}
          </Button>
        </Box>
      </Box>
      <Dialog open={ask} onClose={() => setAsk(false)} transitionDuration={0}>
        <DialogTitle>{t`Grant Flatpak Steam access?`}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {t`This runs the command below once for your user. Steam will then be able to read Mortar's data folder.`}
          </DialogContentText>
          <Box sx={{ mt: 1.5, fontFamily: 'monospace', fontSize: 13, userSelect: 'text' }}>
            {cmd}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAsk(false)}>{t`Cancel`}</Button>
          <Button
            variant="contained"
            onClick={() => {
              setAsk(false)
              GrantSteamAccess().then(load, reportUnexpected)
            }}
          >
            {t`Grant access`}
          </Button>
        </DialogActions>
      </Dialog>
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
  const tellWhenSmapiOut = useSettings((s) => s.tellWhenSmapiOut) !== false
  const push = useToasts((s) => s.push)
  useEffect(() => {
    check(GAME)
    refreshLaunch(GAME)
  }, [check, refreshLaunch])
  const gameVersion = status?.gameVersion ?? ''
  useEffect(() => {
    onVersion(gameVersion)
  }, [gameVersion, onVersion])
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
      <Button
        variant="outlined"
        startIcon={<Download size={16} />}
        disabled={pending || playing}
        onClick={() => install(GAME)}
        sx={{ ...outline, height: 38 }}
      >
        {action}
      </Button>
    )
  }
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.5,
        p: '14px',
        bgcolor: 'rgba(55,55,65,0.9)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <Box
          sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', gap: '2px', minWidth: 0 }}
        >
          <Box sx={{ fontSize: 15, fontWeight: 600 }}>{title}</Box>
          {detail ? (
            <Box
              sx={{
                fontSize: 13,
                color: status?.updateAvailable ? 'warning.main' : 'success.main',
              }}
            >
              {detail}
            </Box>
          ) : null}
        </Box>
        {control}
      </Box>
      <FormControlLabel
        sx={{ m: 0, alignItems: 'center' }}
        control={
          <Switch
            checked={tellWhenSmapiOut}
            onChange={(_, value) =>
              SetTellWhenSmapiOut(value).catch((err: unknown) => {
                const body = errorText(err)
                push({
                  kind: 'error',
                  title: t`Couldn't save that setting`,
                  ...(body ? { body } : {}),
                })
              })
            }
          />
        }
        label={t`Tell me when a new SMAPI is out`}
      />
    </Box>
  )
}

function GameBody() {
  const { t } = useLingui()
  const [folder, setFolder] = useState('')
  const [store, setStore] = useState('')
  const [installs, setInstalls] = useState<FoundInstall[]>([])
  const [version, setVersion] = useState('')
  const load = () => {
    loadGameStatus()
      .then((s) => {
        const g = s.games.find((x) => x.id === GAME)
        setFolder(g?.installDir ?? '')
        setStore(g?.store ?? '')
        setInstalls(g?.installs ?? [])
      })
      .catch(reportUnexpected)
  }
  useEffect(load, [])
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <GameFolder
        folder={folder}
        store={store}
        installs={installs}
        onRefresh={load}
        versionNote={version ? t` · Stardew Valley ${version}` : ''}
      />
      <FlatpakAccess />
      <Smapi key={folder} onVersion={setVersion} />
      <StreamOverlay />
    </Box>
  )
}

export function GameSettings() {
  const override = useSettings((s) => s.gameFolders?.[GAME] ?? '')
  const chosen = useSettings((s) => s.gameStores?.[GAME] ?? '')
  return <GameBody key={`${override}:${chosen}`} />
}
