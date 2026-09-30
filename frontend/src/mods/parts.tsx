import { Trans, useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
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
  Switch,
} from '@mui/material'
import { Ellipsis, FolderOpen, Trash2 } from 'lucide-react'
import { useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { siblingsOf } from './lookup.ts'
import { useMods } from './store.ts'

const paper = { sx: { bgcolor: 'rgba(40,40,48,0.92)' } }

const HASH_MULTIPLIER = 31
const UINT32_BITS = 32
const UINT32_RANGE = 2 ** UINT32_BITS
const HUE_DEGREES = 360
const TILE_FONT_RATIO = 0.5
const DEFAULT_TILE_SIZE = 40

function hash(s: string): number {
  let h = 0
  for (const c of s) {
    h = (h * HASH_MULTIPLIER + (c.codePointAt(0) ?? 0)) % UINT32_RANGE
  }
  return h
}

export function LetterTile({ mod, size = DEFAULT_TILE_SIZE }: { mod: Mod; size?: number }) {
  return (
    <Box
      aria-hidden={true}
      sx={{
        width: size,
        height: size,
        flexShrink: 0,
        borderRadius: '8px',
        display: 'grid',
        placeItems: 'center',
        fontWeight: 700,
        fontSize: size * TILE_FONT_RATIO,
        bgcolor: `hsl(${hash(mod.uniqueId.toLowerCase()) % HUE_DEGREES} 35% 38% / 0.85)`,
      }}
    >
      {(Array.from(mod.name)[0] ?? '?').toUpperCase()}
    </Box>
  )
}

export function ModSwitch({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const setEnabled = useMods((s) => s.setEnabled)
  return (
    <Switch
      checked={mod.enabled}
      onChange={(e) => {
        setEnabled(mod, e.target.checked).catch(reportUnexpected)
      }}
      onClick={(e) => e.stopPropagation()}
      slotProps={{ input: { 'aria-label': t`Enable ${mod.name}` } }}
    />
  )
}

export function ShowFilesButton({ mod }: { mod: Mod }) {
  const showFiles = useMods((s) => s.showFiles)
  return (
    <Button
      startIcon={<FolderOpen size={16} />}
      onClick={() => {
        showFiles(mod).catch(reportUnexpected)
      }}
    >
      <Trans>Show files</Trans>
    </Button>
  )
}

export function RemoveButton({ mod }: { mod: Mod }) {
  const askRemove = useMods((s) => s.askRemove)
  return (
    <Button color="error" startIcon={<Trash2 size={16} />} onClick={() => askRemove(mod)}>
      <Trans>Remove</Trans>
    </Button>
  )
}

export function ModMenu({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const showFiles = useMods((s) => s.showFiles)
  const askRemove = useMods((s) => s.askRemove)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const close = () => setAnchor(null)
  return (
    <>
      <IconButton
        aria-label={t`More actions for ${mod.name}`}
        size="small"
        onClick={(e) => {
          e.stopPropagation()
          setAnchor(e.currentTarget)
        }}
      >
        <Ellipsis size={18} />
      </IconButton>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={close} slotProps={{ paper }}>
        <MenuItem
          onClick={() => {
            close()
            showFiles(mod).catch(reportUnexpected)
          }}
        >
          <ListItemIcon>
            <FolderOpen size={16} />
          </ListItemIcon>
          <ListItemText>
            <Trans>Show files</Trans>
          </ListItemText>
        </MenuItem>
        <MenuItem
          onClick={() => {
            close()
            askRemove(mod)
          }}
        >
          <ListItemIcon>
            <Trash2 size={16} />
          </ListItemIcon>
          <ListItemText>
            <Trans>Remove</Trans>
          </ListItemText>
        </MenuItem>
      </Menu>
    </>
  )
}

export function RemoveDialog() {
  const mod = useMods((s) => s.removing)
  const mods = useMods((s) => s.mods)
  const askRemove = useMods((s) => s.askRemove)
  const remove = useMods((s) => s.remove)
  const others = mod ? siblingsOf(mods, mod).map((m) => m.name) : []
  const close = () => askRemove(null)
  return (
    <Dialog open={mod !== null} onClose={close} slotProps={{ paper }}>
      <DialogTitle>
        <Trans>Remove {mod?.name} from this profile?</Trans>
      </DialogTitle>
      <DialogContent>
        <DialogContentText>
          {others.length > 0 ? (
            <Trans>
              It came in one download with {others.join(', ')}, and all of them are removed
              together.
            </Trans>
          ) : (
            <Trans>Its folder in this profile is deleted.</Trans>
          )}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>
          <Trans>Cancel</Trans>
        </Button>
        <Button
          color="error"
          onClick={() => {
            close()
            if (mod) {
              remove(mod).catch(reportUnexpected)
            }
          }}
        >
          <Trans>Remove</Trans>
        </Button>
      </DialogActions>
    </Dialog>
  )
}
