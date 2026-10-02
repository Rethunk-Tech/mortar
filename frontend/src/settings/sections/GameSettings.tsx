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
import { Clipboard } from '@wailsio/runtime'
import { Copy, Download, FolderOpen, Undo2 } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type { FoundInstall } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import {
  GrantSteamAccess,
  SteamAccess,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/game/service.ts'
import { State } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import {
  ChooseGameFolder,
  SetGameFolder,
  SetGameStore,
  SetTellWhenSmapiOut,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { loadGameStatus } from '../../games/status.ts'
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
  if (store === 'flatpak-steam') {
    return t`Flatpak Steam`
  }
  if (store === 'gog') {
    return t`GOG`
  }
  if (store === 'gog-heroic') {
    return t`GOG via Heroic`
  }
  if (store === 'lutris') {
    return t`Lutris`
  }
  return t`Steam`
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
  const change = (run: Promise<void>) => {
    setError('')
    run
      .then(onRefresh)
      .catch((e: unknown) => setError(errorText(e) ?? t`That folder cannot be used`))
  }
  let foundIn = t`Steam`
  if (store === 'flatpak-steam') {
    foundIn = t`Flatpak Steam`
  } else if (store === 'gog') {
    foundIn = t`GOG`
  } else if (store === 'gog-heroic') {
    foundIn = t`GOG via Heroic`
  } else if (store === 'lutris') {
    foundIn = t`Lutris`
  }
  let source = t`Stardew Valley was not found. Browse to its folder.`
  if (override) {
    source = t`Chosen by you`
  } else if (folder) {
    source = t`Found in ${foundIn}`
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Game folder`}</Box>
      {installs.length > 1 ? (
        <RadioGroup
          value={override ? '' : store}
          onChange={(e) =>
            change(SetGameFolder(GAME, '').then(() => SetGameStore(GAME, e.target.value)))
          }
        >
          {installs.map((item) => (
            <FormControlLabel
              key={`${item.store}:${item.dir}`}
              value={item.store}
              control={<Radio size="small" />}
              label={
                <Box sx={{ display: 'flex', flexDirection: 'column' }}>
                  <StoreLabel store={item.store} />
                  <Box sx={{ fontSize: 12, color: 'rgba(225,225,230,0.95)' }}>{item.dir}</Box>
                </Box>
              }
            />
          ))}
        </RadioGroup>
      ) : null}
      <Box sx={{ display: 'flex', gap: 1 }}>
        <Box
          role="textbox"
          aria-readonly={true}
          aria-label={t`Game folder`}
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
        alignItems: 'center',
        gap: 1.5,
        p: '14px',
        bgcolor: 'rgba(55,55,65,0.9)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column', gap: '2px', minWidth: 0 }}>
        <Box sx={{ fontSize: 15, fontWeight: 600 }}>{title}</Box>
        {detail ? (
          <Box
            sx={{ fontSize: 13, color: status?.updateAvailable ? 'warning.main' : 'success.main' }}
          >
            {detail}
          </Box>
        ) : null}
      </Box>
      {control}
    </Box>
  )
}

// SMAPI is Stardew's loader, so whether Mortar announces a new SMAPI belongs with the game, not the global Updates.
function SmapiNotice() {
  const { t } = useLingui()
  const on = useSettings((s) => s.tellWhenSmapiOut) !== false
  const push = useToasts((s) => s.push)
  return (
    <FormControlLabel
      sx={{ m: 0, alignItems: 'center' }}
      control={
        <Switch
          checked={on}
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
      <SmapiNotice />
      <StreamOverlay />
    </Box>
  )
}

export function GameSettings() {
  const override = useSettings((s) => s.gameFolders?.[GAME] ?? '')
  const chosen = useSettings((s) => s.gameStores?.[GAME] ?? '')
  return <GameBody key={`${override}:${chosen}`} />
}
