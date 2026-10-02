import { useLingui } from '@lingui/react/macro'
import { Box, Button, InputAdornment, TextField, Typography } from '@mui/material'
import { Check, FolderOpen, RefreshCw, TriangleAlert, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  GameInfo,
  StoreApp,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/game/models.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import { SetGameFolder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { gameArt } from '../games/art.ts'
import { useLoader } from '../loader/store.ts'
import { errorMessage } from '../toasts/report.ts'
import { Panel } from './Panel.tsx'
import { useRefreshOnFocus } from './useRefreshOnFocus.ts'

const shadow = '0 1px 2px rgba(0,0,0,0.9), 0 0 18px rgba(0,0,0,0.85)'
// The art is dimmed until the game is found.
const artOpacity = { found: 0.8, missing: 0.4 }

function Hero({ game, dim }: { game: GameInfo; dim?: boolean }) {
  const art = gameArt(game)
  return (
    <Box
      sx={{
        position: 'relative',
        height: 150,
        borderRadius: '6px',
        overflow: 'hidden',
        bgcolor: 'background.paper',
        display: 'flex',
        alignItems: 'flex-end',
        px: '18px',
        pb: '14px',
      }}
    >
      {art ? (
        <Box
          component="img"
          src={art}
          alt=""
          sx={{
            position: 'absolute',
            inset: 0,
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            opacity: dim ? artOpacity.missing : artOpacity.found,
            filter: dim ? 'grayscale(0.6)' : 'none',
          }}
        />
      ) : null}
      <Typography
        sx={{ position: 'relative', fontSize: 34, fontWeight: 600, textShadow: shadow }}
        noWrap={true}
      >
        {game.name}
      </Typography>
    </Box>
  )
}

function storeCaption(store: string, t: (s: TemplateStringsArray) => string): string {
  switch (store) {
    case 'flatpak-steam':
      return t`Found in Steam (Flatpak)`
    case 'gog':
      return t`Found in GOG`
    case 'gog-heroic':
      return t`Found in Heroic`
    case 'lutris':
      return t`Found in Lutris`
    case 'steam':
      return t`Found in Steam`
    default:
      return t`Folder chosen by you`
  }
}

// Where Mortar looked: each launcher, whether it was found, and whether it holds this game.
function Looked({ game, launchers }: { game: GameInfo; launchers: StoreApp[] }) {
  const { t } = useLingui()
  return (
    <Box component="ul" sx={{ m: 0, p: 0, listStyle: 'none', display: 'grid', gap: '6px' }}>
      {launchers.map((l) => {
        const has = (l.games ?? []).some((g) => g.id === game.id)
        let line = t`not found`
        if (l.found && !has) {
          line = t`found, without ${game.name}`
        } else if (has) {
          line = t`has ${game.name}`
        }
        return (
          <Box
            component="li"
            key={l.id}
            sx={{ display: 'flex', alignItems: 'center', gap: 1, fontSize: 14 }}
          >
            {l.found ? <Check size={15} color="#0CDF64" /> : <X size={15} color="#9AA0AA" />}
            <Box component="span" sx={{ fontWeight: 600 }}>
              {l.name}
            </Box>
            <Box component="span" sx={{ color: 'text.secondary' }}>
              {line}
            </Box>
          </Box>
        )
      })}
    </Box>
  )
}

export function FindStep({
  game,
  launchers,
  refresh,
  onContinue,
}: {
  game: GameInfo
  launchers: StoreApp[]
  refresh: () => void
  onContinue: () => void
}) {
  const { t } = useLingui()
  const found = game.installed
  const smapi = useLoader((s) => s.status)
  const check = useLoader((s) => s.check)
  const [error, setError] = useState('')
  const dir = game.installDir
  useRefreshOnFocus(refresh, !found)
  useEffect(() => {
    if (dir) {
      check(game.id)
    }
  }, [dir, check, game.id])

  const browse = async () => {
    try {
      const picked = await PickFolder(t`Choose your ${game.name} folder`)
      if (!picked) {
        return
      }
      await SetGameFolder(game.id, picked)
      setError('')
      refresh()
    } catch (e) {
      setError(errorMessage(e))
    }
  }

  const details = [
    smapi?.gameVersion ? `${game.name} ${smapi.gameVersion}` : '',
    smapi?.installed
      ? t`${game.loader} ${smapi.version} installed`
      : t`${game.loader} not installed yet`,
  ]
    .filter(Boolean)
    .join(' · ')

  return (
    <Panel width={640}>
      <Hero game={game} dim={!found} />
      {found ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <Typography sx={{ fontSize: 13 }}>{storeCaption(game.store, t)}</Typography>
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
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<FolderOpen size={16} />}
              onClick={browse}
              sx={{ whiteSpace: 'nowrap', height: 40, flexShrink: 0 }}
            >
              {t`Change…`}
            </Button>
          </Box>
          <Typography sx={{ fontSize: 13 }}>{details}</Typography>
        </Box>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25 }}>
            <TriangleAlert size={18} color="#F3B416" />
            <Typography sx={{ fontSize: 16, fontWeight: 600 }}>
              {t`${game.name} was not found in your launchers`}
            </Typography>
          </Box>
          <Looked game={game} launchers={launchers} />
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Button
              variant="contained"
              startIcon={<FolderOpen size={16} />}
              onClick={browse}
              sx={{ whiteSpace: 'nowrap', height: 40 }}
            >
              {t`Choose the game folder…`}
            </Button>
            <Button
              variant="text"
              color="inherit"
              startIcon={<RefreshCw size={15} />}
              onClick={refresh}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {t`Rescan`}
            </Button>
          </Box>
        </Box>
      )}
      {error ? (
        <Typography role="alert" sx={{ fontSize: 13, color: 'error.light' }}>
          {error}
        </Typography>
      ) : null}
      {found ? (
        <Button
          variant="contained"
          onClick={onContinue}
          sx={{ height: 46, fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}
        >
          {t`Continue`}
        </Button>
      ) : null}
    </Panel>
  )
}
