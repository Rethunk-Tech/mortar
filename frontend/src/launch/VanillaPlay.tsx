import { useLingui } from '@lingui/react/macro'
import {
  Button,
  ButtonGroup,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
} from '@mui/material'
import { ChevronDown, Gamepad2, Play } from 'lucide-react'
import { type MouseEvent, useState } from 'react'
import { ForcesSMAPI } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { compact } from '../game/compact.ts'
import { SmapiWarnDialog } from './SmapiWarnDialog.tsx'
import { useLaunch } from './store.ts'

export function VanillaPlay({
  game,
  playDisabled,
  vanillaDisabled,
  label,
  play,
}: {
  game: string
  playDisabled: boolean
  vanillaDisabled: boolean
  label: string
  play: () => void
}) {
  const { t } = useLingui()
  const startVanilla = useLaunch((s) => s.startVanilla)
  const [menu, setMenu] = useState<HTMLElement | null>(null)
  const [smapiWarn, setSmapiWarn] = useState(false)
  const openMenu = (el: HTMLElement) => {
    if (!vanillaDisabled) {
      setMenu(el)
    }
  }
  const onContext = (e: MouseEvent<HTMLElement>) => {
    e.preventDefault()
    openMenu(e.currentTarget)
  }
  const playVanilla = () => {
    setMenu(null)
    ForcesSMAPI(game)
      .then((forces) => {
        if (forces) {
          setSmapiWarn(true)
          return
        }
        return startVanilla(game, false)
      })
      .catch(() => startVanilla(game, false))
  }
  return (
    <>
      <ButtonGroup
        variant="contained"
        fullWidth={true}
        sx={{
          height: 58,
          borderRadius: 0,
          boxShadow: 'none',
          [compact]: { display: 'none' },
          '& .MuiButtonGroup-grouped': { minWidth: 0 },
        }}
      >
        <Button
          disabled={playDisabled}
          startIcon={<Play size={22} fill="currentColor" />}
          onClick={play}
          onContextMenu={onContext}
          sx={{
            flex: 1,
            height: 58,
            borderRadius: 0,
            fontSize: 22,
            fontWeight: 700,
            textTransform: 'none',
            boxShadow: 'none',
            whiteSpace: 'nowrap',
            '& .MuiButton-startIcon': { mr: '10px' },
          }}
        >
          {label}
        </Button>
        <Button
          disabled={vanillaDisabled}
          aria-label={t`More play options`}
          onClick={(e) => openMenu(e.currentTarget)}
          sx={{ width: 40, minWidth: 40, px: 0, height: 58, borderRadius: 0, boxShadow: 'none' }}
        >
          <ChevronDown size={18} />
        </Button>
      </ButtonGroup>
      <IconButton
        aria-label={label}
        title={label}
        disabled={playDisabled}
        onClick={play}
        onContextMenu={onContext}
        sx={{
          display: 'none',
          borderRadius: 0,
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
          '&:hover': { bgcolor: 'primary.dark' },
          [compact]: { display: 'flex', width: '100%', height: 50 },
        }}
      >
        <Play size={20} fill="currentColor" />
      </IconButton>
      <Menu anchorEl={menu} open={menu !== null} onClose={() => setMenu(null)}>
        <MenuItem disabled={vanillaDisabled} onClick={playVanilla}>
          <ListItemIcon>
            <Gamepad2 size={16} />
          </ListItemIcon>
          <ListItemText>{t`Play without mods`}</ListItemText>
        </MenuItem>
      </Menu>
      <SmapiWarnDialog
        open={smapiWarn}
        onClose={() => setSmapiWarn(false)}
        onPlay={() => {
          setSmapiWarn(false)
          startVanilla(game, false).then(() => undefined)
        }}
      />
    </>
  )
}
