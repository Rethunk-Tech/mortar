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
import { compact } from '../game/compact.ts'
import { routeGame, useNav } from '../nav/store.ts'
import type { PlayPreset } from '../profiles/profilePresets.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { playMenuEntries } from './playMenu.ts'
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
      <ConfirmDialog
        open={linuxDirect}
        maxWidth={440}
        title={t`Play without Steam overlay`}
        body={t`Mortar starts the unmodded game directly, skipping SMAPI's Linux launcher. The Steam overlay and playtime tracking do not work; Steam still supplies its API if running.`}
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
  play,
  presets,
  playWith,
  setDefault,
}: {
  game: string
  playDisabled: boolean
  vanillaDisabled: boolean
  label: string
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
            '& .MuiButton-startIcon': { mr: '10px' },
          }}
        >
          {label}
        </Button>
        <Button
          disabled={vanillaDisabled}
          aria-label={t`More play options`}
          aria-haspopup="menu"
          aria-expanded={menu !== null}
          onClick={(e) => openMenu(e.currentTarget)}
          sx={{ width: 40, minWidth: 40, px: 0, height: 58, borderRadius: 0 }}
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
                <ListItemText
                  secondary={preset?.isDefault ? t`Default` : undefined}
                >{t`Play with ${preset?.base ? t`Standard` : preset?.name}`}</ListItemText>
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
          return (
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
            </MenuItem>
          )
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
