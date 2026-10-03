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
import { ChevronDown, Gamepad2, Play } from 'lucide-react'
import { type MouseEvent, useState } from 'react'
import { compact } from '../game/compact.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { playVanillaOpen, rememberLinuxVanillaDirect, useVanillaPrompt } from './playOpen.ts'
import { SmapiWarnDialog } from './SmapiWarnDialog.tsx'
import { useLaunch } from './store.ts'

export function VanillaPlayDialogs() {
  const { t } = useLingui()
  const game = routeGame(useNav((s) => s.route))
  const startVanilla = useLaunch((s) => s.startVanilla)
  const smapiWarn = useVanillaPrompt((s) => s.smapiWarn)
  const linuxDirect = useVanillaPrompt((s) => s.linuxDirect)
  const setSmapiWarn = useVanillaPrompt((s) => s.setSmapiWarn)
  const setLinuxDirect = useVanillaPrompt((s) => s.setLinuxDirect)
  if (!game) {
    return null
  }
  return (
    <>
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

export function VanillaPlay({
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
  const [menu, setMenu] = useState<HTMLElement | null>(null)
  const openMenu = (el: HTMLElement) => {
    if (!vanillaDisabled) {
      setMenu(el)
    }
  }
  const onContext = (e: MouseEvent<HTMLElement>) => {
    e.preventDefault()
    openMenu(e.currentTarget)
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
        <MenuItem
          disabled={vanillaDisabled}
          onClick={() => {
            setMenu(null)
            playVanillaOpen()
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <Gamepad2 size={16} />
          </ListItemIcon>
          <ListItemText>{t`Play without mods`}</ListItemText>
        </MenuItem>
      </Menu>
    </>
  )
}
