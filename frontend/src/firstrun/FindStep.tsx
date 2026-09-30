import { useLingui } from '@lingui/react/macro'
import { Box, Button, InputAdornment, TextField, Typography } from '@mui/material'
import { Check, FolderOpen, RefreshCw, TriangleAlert } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { GameInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import { SetGameFolder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import type { GameStatus } from '../games/status.ts'
import { useLoader } from '../loader/store.ts'
import { errorText } from '../toasts/report.ts'
import { STARDEW } from './needed.ts'
import { Panel } from './Panel.tsx'

const shadow = '0 1px 2px rgba(0,0,0,0.9), 0 0 18px rgba(0,0,0,0.85)'

function Hero({ game }: { game: GameInfo | undefined }) {
  return (
    <Box
      sx={{
        position: 'relative',
        height: 150,
        borderRadius: '6px',
        overflow: 'hidden',
        bgcolor: 'rgba(0,0,0,0.35)',
        display: 'flex',
        alignItems: 'center',
        px: '18px',
      }}
    >
      {game?.artUrl ? (
        <Box
          component="img"
          src={game.artUrl}
          alt=""
          sx={{
            position: 'absolute',
            inset: 0,
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            opacity: 0.72,
          }}
        />
      ) : null}
      <Typography
        sx={{ position: 'relative', fontSize: 34, fontWeight: 600, textShadow: shadow }}
        noWrap={true}
      >
        {game?.name ?? 'Stardew Valley'}
      </Typography>
    </Box>
  )
}

function BrowseButton({ onClick }: { onClick: () => void }) {
  const { t } = useLingui()
  return (
    <Button
      variant="outlined"
      color="inherit"
      startIcon={<FolderOpen size={16} />}
      onClick={onClick}
      sx={{ whiteSpace: 'nowrap', height: 40, flexShrink: 0 }}
    >
      {t`Browse…`}
    </Button>
  )
}

export function FindStep({
  status,
  refresh,
  onContinue,
}: {
  status: GameStatus
  refresh: () => void
  onContinue: () => void
}) {
  const { t } = useLingui()
  const game = status.games.find((g) => g.id === STARDEW)
  const found = game?.installed === true
  const smapi = useLoader((s) => s.status)
  const check = useLoader((s) => s.check)
  const [error, setError] = useState('')
  const dir = game?.installDir ?? ''
  useEffect(() => {
    if (dir) {
      check(STARDEW)
    }
  }, [dir, check])

  const browse = async () => {
    try {
      const picked = await PickFolder(t`Choose your Stardew Valley folder`)
      if (!picked) {
        return
      }
      await SetGameFolder(STARDEW, picked)
      setError('')
      refresh()
    } catch (e) {
      setError(errorText(e) ?? String(e))
    }
  }

  const details = [
    smapi?.gameVersion ? t`Stardew Valley ${smapi.gameVersion}` : '',
    smapi?.installed ? t`SMAPI ${smapi.version} installed` : t`SMAPI not installed yet`,
  ]
    .filter(Boolean)
    .join(' · ')

  return (
    <Panel width={640}>
      <Hero game={game} />
      {found ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <Typography sx={{ fontSize: 13 }}>{t`Found in your Steam library`}</Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <TextField
              value={dir}
              size="small"
              fullWidth={true}
              slotProps={{
                htmlInput: { readOnly: true, 'aria-label': t`Game folder` },
                input: {
                  startAdornment: (
                    <InputAdornment position="start">
                      <Check size={16} color="#0CDF64" />
                    </InputAdornment>
                  ),
                },
                root: { sx: { userSelect: 'text' } },
              }}
            />
            <BrowseButton onClick={browse} />
          </Box>
          <Typography sx={{ fontSize: 13 }}>{details}</Typography>
        </Box>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
            <TriangleAlert size={18} color="#F3B416" />
            <Typography sx={{ fontSize: 16, fontWeight: 600 }}>
              {t`Stardew Valley was not found`}
            </Typography>
          </Box>
          <Typography sx={{ fontSize: 14 }}>
            {t`Mortar supports Steam installed directly on the system.`}{' '}
            {status.steam === 'flatpak-only'
              ? t`Only a Flatpak Steam was found.`
              : t`Steam was not found.`}{' '}
            {t`Choose the folder that holds Stardew Valley instead.`}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <BrowseButton onClick={browse} />
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<RefreshCw size={16} />}
              onClick={refresh}
              sx={{ whiteSpace: 'nowrap', height: 40 }}
            >
              {t`Retry`}
            </Button>
          </Box>
        </Box>
      )}
      {error ? (
        <Typography role="alert" sx={{ fontSize: 13, color: 'error.light' }}>
          {error}
        </Typography>
      ) : null}
      <Button
        variant="contained"
        disabled={!found}
        onClick={onContinue}
        sx={{ height: 46, fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}
      >
        {t`Continue`}
      </Button>
    </Panel>
  )
}
