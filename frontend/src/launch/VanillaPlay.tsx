import { useLingui } from '@lingui/react/macro'
import {
  Button,
  ButtonGroup,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Tooltip,
} from '@mui/material'
import { System } from '@wailsio/runtime'
import { ChevronDown, Gamepad2, Play } from 'lucide-react'
import { type MouseEvent, useState } from 'react'
import { ForcesSMAPI } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/service.ts'
import { compact } from '../game/compact.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { windowsVanillaAfterForcesCheck } from './playOpen.ts'
import { SmapiWarnDialog } from './SmapiWarnDialog.tsx'
import { useLaunch } from './store.ts'

const linuxVanillaDirectKey = 'mortar.linuxVanillaDirect'

function linuxVanillaDirectAgreed(): boolean {
  try {
    return localStorage.getItem(linuxVanillaDirectKey) === '1'
  } catch {
    return false
  }
}

function rememberLinuxVanillaDirect() {
  try {
    localStorage.setItem(linuxVanillaDirectKey, '1')
  } catch {
    // Private mode can refuse localStorage; the next Play without mods asks again.
  }
}

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
  const [linuxDirect, setLinuxDirect] = useState(false)
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
    if (!System.IsWindows()) {
      if (!linuxVanillaDirectAgreed()) {
        setLinuxDirect(true)
        return
      }
      startVanilla(game, true).catch(() => undefined)
      return
    }
    ForcesSMAPI(game)
      .then((forces) => {
        const next = windowsVanillaAfterForcesCheck(true, forces)
        if (next === 'warn') {
          setSmapiWarn(true)
          return
        }
        return startVanilla(game, false)
      })
      .catch((e: unknown) => {
        reportUnexpected(e)
      })
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
      <Tooltip title={label}>
        <span>
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
        </span>
      </Tooltip>
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
      <Dialog
        open={linuxDirect}
        onClose={() => setLinuxDirect(false)}
        transitionDuration={0}
        slotProps={{ paper: { sx: { maxWidth: 440 } } }}
      >
        <DialogTitle>{t`Play without Steam overlay`}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {t`Mortar starts the unmodded game directly so SMAPI's Linux launcher is not used. The Steam overlay and Steam's playtime tracking will not work. Steam still supplies the API if it is running.`}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setLinuxDirect(false)} sx={{ whiteSpace: 'nowrap' }}>
            {t`Cancel`}
          </Button>
          <Button
            variant="contained"
            sx={{ whiteSpace: 'nowrap' }}
            onClick={() => {
              rememberLinuxVanillaDirect()
              setLinuxDirect(false)
              startVanilla(game, true).then(() => undefined)
            }}
          >
            {t`Play without mods`}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
