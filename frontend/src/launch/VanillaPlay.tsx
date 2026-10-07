import { useLingui } from '@lingui/react/macro'
import {
  Button,
  ButtonGroup,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControlLabel,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Radio,
  RadioGroup,
  Tooltip,
} from '@mui/material'
import { ChevronDown, Gamepad2, Play, Star } from 'lucide-react'
import { type MouseEvent, useState } from 'react'
import { MenuHeading } from '../game/MenuHeading.tsx'
import { routeGame, useNav } from '../nav/store.ts'
import type { PlayPreset } from '../profiles/profilePresets.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { PLAY_HEIGHT_PX } from './playHeight.ts'
import { playMenuEntries } from './playMenu.ts'
import { playVanillaOpen, rememberLinuxVanillaDirect, useVanillaPrompt } from './playOpen.ts'
import { SmapiWarnDialog } from './SmapiWarnDialog.tsx'
import { useLaunch } from './store.ts'

// Play with its menu caret, or on the narrow rail Play alone as an icon (its menu then opens on right-click).
function PlayButtons({
  rail,
  label,
  playDisabled,
  vanillaDisabled,
  menuOpen,
  onPlay,
  onMenu,
  onContext,
}: {
  rail: boolean
  label: string
  playDisabled: boolean
  vanillaDisabled: boolean
  menuOpen: boolean
  onPlay: () => void
  onMenu: (el: HTMLElement) => void
  onContext: (e: MouseEvent<HTMLElement>) => void
}) {
  const { t } = useLingui()
  return rail ? (
    // The button carries its own name; a plain span may not take the label Tooltip would give it.
    <Tooltip title={label} describeChild={true}>
      <span style={{ display: 'flex' }}>
        <IconButton
          aria-label={label}
          title={label}
          disabled={playDisabled}
          onClick={onPlay}
          onContextMenu={onContext}
          sx={{
            display: 'flex',
            width: '100%',
            height: PLAY_HEIGHT_PX,
            borderRadius: 0,
            bgcolor: 'primary.main',
            color: 'primary.contrastText',
            '&:hover': { bgcolor: 'primary.dark' },
          }}
        >
          <Play size={28} fill="currentColor" />
        </IconButton>
      </span>
    </Tooltip>
  ) : (
    <ButtonGroup
      variant="contained"
      fullWidth={true}
      sx={{
        height: PLAY_HEIGHT_PX,
        borderRadius: 0,
        boxShadow: 'none',
        '& .MuiButtonGroup-grouped': { minWidth: 0 },
      }}
    >
      <Button
        disabled={playDisabled}
        startIcon={<Play size={20} fill="currentColor" />}
        onClick={onPlay}
        onContextMenu={onContext}
        sx={{
          flex: 1,
          height: PLAY_HEIGHT_PX,
          borderRadius: 0,
          fontSize: 20,
          fontWeight: 700,
          '& .MuiButton-startIcon': { mr: '10px' },
        }}
      >
        {label}
      </Button>
      <Button
        disabled={vanillaDisabled}
        aria-label={t`More play options`}
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        onClick={(e) => onMenu(e.currentTarget)}
        sx={{ width: 40, minWidth: 40, px: 0, height: PLAY_HEIGHT_PX, borderRadius: 0 }}
      >
        <ChevronDown size={18} />
      </Button>
    </ButtonGroup>
  )
}

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
      <ConfirmDialog
        open={linuxDirect}
        maxWidth={440}
        title={t`Play without Steam overlay`}
        body={t`Mortar starts the unmodded game directly, outside Steam. The Steam overlay and playtime tracking do not work; Steam still supplies its API if running.`}
        confirmLabel={t`Play without mods`}
        onCancel={() => setLinuxDirect(false)}
        onConfirm={() => {
          rememberLinuxVanillaDirect()
          setLinuxDirect(false)
          startVanilla(game, true).then(() => undefined)
        }}
      />
    </>
  )
}

export function VanillaPlay({
  playDisabled,
  vanillaDisabled,
  label,
  rail,
  play,
  presets,
  playWith,
  setDefault,
}: {
  game: string
  playDisabled: boolean
  vanillaDisabled: boolean
  label: string
  rail: boolean
  play: () => void
  presets: PlayPreset[]
  playWith: (key: string) => void
  setDefault: (key: string) => void
}) {
  const { t } = useLingui()
  const [menu, setMenu] = useState<HTMLElement | null>(null)
  const [choosing, setChoosing] = useState(false)
  const entries = playMenuEntries(presets)
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
      <PlayButtons
        rail={rail}
        label={label}
        playDisabled={playDisabled}
        vanillaDisabled={vanillaDisabled}
        menuOpen={menu !== null}
        onPlay={play}
        onMenu={openMenu}
        onContext={onContext}
      />
      <Menu anchorEl={menu} open={menu !== null} onClose={() => setMenu(null)}>
        {presets.length > 0 ? <MenuHeading>{t`Play with`}</MenuHeading> : null}
        {entries.map((entry) => {
          if (entry.kind === 'play') {
            const preset = presets.find((p) => p.key === entry.key)
            return (
              <MenuItem
                key={entry.key}
                disabled={playDisabled}
                onClick={() => {
                  setMenu(null)
                  playWith(entry.key)
                }}
              >
                <ListItemIcon sx={{ color: 'inherit' }}>
                  <Play size={16} />
                </ListItemIcon>
                <ListItemText>
                  {preset?.base ? t`Standard` : preset?.name}
                  {preset?.isDefault ? ` (${t`Default`})` : ''}
                </ListItemText>
              </MenuItem>
            )
          }
          if (entry.kind === 'setDefault') {
            return [
              <Divider key="divider" />,
              <MenuItem
                key="setDefault"
                onClick={() => {
                  setMenu(null)
                  setChoosing(true)
                }}
              >
                <ListItemIcon sx={{ color: 'inherit' }}>
                  <Star size={16} />
                </ListItemIcon>
                <ListItemText>{t`Set default preset…`}</ListItemText>
              </MenuItem>,
            ]
          }
          return [
            presets.length > 0 ? <Divider key="vanilla-divider" /> : null,
            <MenuItem
              key="vanilla"
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
            </MenuItem>,
          ]
        })}
      </Menu>
      <Dialog open={choosing} onClose={() => setChoosing(false)}>
        <DialogTitle>{t`Default launch preset`}</DialogTitle>
        <DialogContent>
          <RadioGroup
            value={presets.find((p) => p.isDefault)?.key ?? ''}
            onChange={(e) => {
              setChoosing(false)
              setDefault(e.target.value)
            }}
          >
            {presets.map((p) => (
              <FormControlLabel
                key={p.key}
                value={p.key}
                control={<Radio />}
                label={p.base ? t`Standard` : p.name}
              />
            ))}
          </RadioGroup>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setChoosing(false)}>{t`Cancel`}</Button>
        </DialogActions>
      </Dialog>
    </>
  )
}
