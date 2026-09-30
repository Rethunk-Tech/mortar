import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Download, FolderOpen, Undo2 } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import {
  ChooseGameFolder,
  SetGameFolder,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { loadGameStatus } from '../../games/status.ts'
import { InstallSteps } from '../../loader/InstallSteps.tsx'
import { useLoader } from '../../loader/store.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useSettings } from '../store.ts'

const GAME = 'stardew'

const outline = { whiteSpace: 'nowrap', flexShrink: 0, height: 42 }

function GameFolder({ folder, versionNote }: { folder: string; versionNote: string }) {
  const { t } = useLingui()
  const override = useSettings((s) => s.gameFolders?.[GAME] ?? '')
  const [error, setError] = useState('')
  const change = (run: Promise<void>) => {
    setError('')
    run.catch((e: unknown) => setError(errorText(e) ?? t`That folder cannot be used`))
  }
  let source = t`Stardew Valley was not found in Steam. Browse to its folder.`
  if (override) {
    source = t`Chosen by you`
  } else if (folder) {
    source = t`Found in Steam`
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Game folder`}</Box>
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
            {t`Use Steam's`}
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

function Smapi({ onVersion }: { onVersion: (v: string) => void }) {
  const { t } = useLingui()
  const status = useLoader((s) => s.status)
  const installing = useLoader((s) => s.installing)
  const steps = useLoader((s) => s.steps)
  const check = useLoader((s) => s.check)
  const install = useLoader((s) => s.install)
  useEffect(() => {
    check(GAME)
  }, [check])
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

function GameBody() {
  const [folder, setFolder] = useState('')
  const [version, setVersion] = useState('')
  useEffect(() => {
    let live = true
    loadGameStatus()
      .then((s) => {
        if (live) {
          setFolder(s.games.find((g) => g.id === GAME)?.installDir ?? '')
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [])
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <GameFolder folder={folder} versionNote={version ? ` · Stardew Valley ${version}` : ''} />
      <Smapi key={folder} onVersion={setVersion} />
    </Box>
  )
}

// Remounting on an override change re-discovers the folder and re-checks SMAPI against it.
export function GameSettings() {
  const override = useSettings((s) => s.gameFolders?.[GAME] ?? '')
  return <GameBody key={override} />
}
