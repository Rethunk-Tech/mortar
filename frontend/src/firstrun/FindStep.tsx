import { useLingui } from '@lingui/react/macro'
import { Box, Button, InputAdornment, TextField, Typography } from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { Check, FolderOpen, RefreshCw, TriangleAlert, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  GameInfo,
  StoreApp,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/models.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetGameFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { gameArt } from '../games/art.ts'
import { storeName } from '../games/storeName.ts'
import { useLoader } from '../loader/store.ts'
import { TypedFolder } from '../shell/TypedFolder.tsx'
import { space } from '../theme/density.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { Panel } from './Panel.tsx'
import { useRefreshOnFocus } from './useRefreshOnFocus.ts'

const shadow = '0 1px 2px var(--mortar-overlay-90), 0 0 18px var(--mortar-overlay-85)'
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
        title={game.name}
        sx={{ position: 'relative', fontSize: 34, fontWeight: 600, textShadow: shadow }}
        noWrap={true}
      >
        {game.name}
      </Typography>
    </Box>
  )
}

// Where Mortar looked: each launcher, whether it was found, and whether it holds this game.
function Looked({ game, launchers }: { game: GameInfo; launchers: StoreApp[] }) {
  const { t } = useLingui()
  const { success, text } = useTheme().palette
  const ok = success.main
  const muted = text.secondary
  return (
    <Box component="ul" sx={{ m: 0, p: 0, listStyle: 'none', display: 'grid', gap: '6px' }}>
      {launchers.map((l) => {
        const has = (l.games ?? []).some((g) => g.id === game.id)
        let line = t`Not found`.toLowerCase()
        if (l.found && !has) {
          line = t`found, without ${game.name}`
        } else if (has) {
          line = t`has ${game.name}`
        }
        return (
          <Box
            component="li"
            key={l.id}
            sx={{ display: 'flex', alignItems: 'center', gap: space.gap, fontSize: 14 }}
          >
            {l.found ? <Check size={15} color={ok} /> : <X size={15} color={muted} />}
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
  const { success, warning } = useTheme().palette
  const found = game.installed
  const smapi = useLoader((s) => s.status)
  const check = useLoader((s) => s.check)
  const [error, setError] = useState<InlineError | null>(null)
  const [editing, setEditing] = useState(false)
  const [typing, setTyping] = useState(false)
  const dir = game.installDir
  const named = storeName(game.store)
  const caption = named ? t`Found in ${{ path: t(named) }}` : t`Folder chosen by you`
  useRefreshOnFocus(refresh, !found)
  useEffect(() => {
    if (dir) {
      check(game.id)
    }
  }, [dir, check, game.id])

  const use = async (chosen: string) => {
    try {
      await SetGameFolder(game.id, chosen)
      setError(null)
      setTyping(false)
      refresh()
    } catch (e) {
      setError(inlineError(e))
    }
  }
  // Without a native folder dialog (server mode) the path is typed instead.
  const browse = () => {
    PickFolder(t`Choose your ${game.name} folder`)
      .then((picked) => (picked ? use(picked) : undefined))
      .catch(() => setTyping(true))
  }

  const details = [
    smapi?.gameVersion ? `${game.name} ${smapi.gameVersion}` : '',
    smapi?.installed
      ? t`${{ loader: game.loader }} ${{ version: smapi.version }} installed`
      : t`${game.loader} not installed yet`,
  ]
    .filter(Boolean)
    .join(' · ')

  return (
    <Panel width={640}>
      <Hero game={game} dim={!found} />
      {found ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <Typography sx={{ fontSize: 13 }}>{caption}</Typography>
          <Box sx={{ display: 'flex', gap: space.gap }}>
            <TextField
              // Unfocused, the path is laid out right to left so a long one loses its start, not the game's folder
              // name; the mark keeps a leading slash at the start. Focused, it is the plain path, to select and copy.
              value={editing ? dir : `\u200e${dir}`}
              onFocus={() => setEditing(true)}
              onBlur={() => setEditing(false)}
              size="small"
              fullWidth={true}
              slotProps={{
                htmlInput: {
                  readOnly: true,
                  'aria-label': t`Game folder`,
                  title: dir,
                  style: editing ? {} : { direction: 'rtl', textOverflow: 'ellipsis' },
                },
                input: {
                  startAdornment: (
                    <InputAdornment position="start">
                      <Check size={16} color={success.main} />
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
              sx={{ height: 40, flexShrink: 0 }}
            >
              {t`Change folder…`}
            </Button>
          </Box>
          <Typography sx={{ fontSize: 13 }}>{details}</Typography>
        </Box>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <TriangleAlert size={18} color={warning.main} />
            <Typography sx={{ fontSize: 16, fontWeight: 600 }}>
              {t`${game.name} was not found in your launchers`}
            </Typography>
          </Box>
          <Looked game={game} launchers={launchers} />
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Button
              variant="contained"
              startIcon={<FolderOpen size={16} />}
              onClick={browse}
              sx={{ height: 40 }}
            >
              {t`Choose folder…`}
            </Button>
            <Button
              variant="text"
              color="inherit"
              startIcon={<RefreshCw size={15} />}
              onClick={refresh}
            >
              {t`Rescan`}
            </Button>
          </Box>
        </Box>
      )}
      {typing ? <TypedFolder label={t`${game.name} folder`} onUse={use} /> : null}
      {error ? (
        <Typography role="alert" title={error.details} sx={{ fontSize: 13, color: 'error.light' }}>
          {error.message}
        </Typography>
      ) : null}
      {found ? (
        <Button variant="contained" onClick={onContinue} size="large">
          {t`Continue`}
        </Button>
      ) : null}
    </Panel>
  )
}
