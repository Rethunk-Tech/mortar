import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Switch,
  Tooltip,
} from '@mui/material'
import { ArrowUp, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDescribe } from './describe.ts'
import { concerns, problemsOf, siblingsOf, updateFor } from './lookup.ts'
import { paper } from './paper.ts'
import { useMods } from './store.ts'
import { useUpdates } from './updates.ts'

const HASH_MULTIPLIER = 31
const UINT32_BITS = 32
const UINT32_RANGE = 2 ** UINT32_BITS
const HUE_DEGREES = 360
const TILE_FONT_RATIO = 0.5
const DEFAULT_TILE_SIZE = 48
const SMALL_TILE = 32

function hash(s: string): number {
  let h = 0
  for (const c of s) {
    h = (h * HASH_MULTIPLIER + (c.codePointAt(0) ?? 0)) % UINT32_RANGE
  }
  return h
}

// The mod's Nexus picture when it has one that loads, else its first letter on a colour from its UniqueID.
export function LetterTile({
  mod,
  size = DEFAULT_TILE_SIZE,
}: {
  mod: Pick<Mod, 'uniqueId' | 'name'> & { picture?: string }
  size?: number
}) {
  const [failed, setFailed] = useState<string | null>(null)
  const picture = mod.picture && failed !== mod.picture ? mod.picture : ''
  return (
    <Box
      aria-hidden={true}
      className="tile"
      sx={{
        width: size,
        height: size,
        flexShrink: 0,
        borderRadius: size < SMALL_TILE ? '4px' : '6px',
        display: 'grid',
        placeItems: 'center',
        overflow: 'hidden',
        fontWeight: 700,
        fontSize: size * TILE_FONT_RATIO,
        bgcolor: `hsl(${hash(mod.uniqueId.toLowerCase()) % HUE_DEGREES} 35% 38% / 0.85)`,
      }}
    >
      {picture ? (
        <Box
          component="img"
          alt=""
          src={picture}
          onError={() => setFailed(picture)}
          sx={{ width: '100%', height: '100%', objectFit: 'cover' }}
        />
      ) : (
        (Array.from(mod.name)[0] ?? '?').toUpperCase()
      )}
    </Box>
  )
}

export function ProblemBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const describe = useDescribe()
  const result = useMods((s) => s.problems)
  const mine = problemsOf(result).filter((p) => concerns(p, mod))
  if (mine.length === 0) {
    return null
  }
  const text = mine.map(describe).join(' ')
  return (
    <Tooltip title={text}>
      <Box
        role="img"
        aria-label={t`Problem: ${text}`}
        sx={{ display: 'flex', flexShrink: 0, color: 'warning.main' }}
      >
        <TriangleAlert size={16} />
      </Box>
    </Tooltip>
  )
}

export function UpdateBadge({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const update = useUpdates((s) => updateFor(s.updates, mod))
  if (!update) {
    return null
  }
  const text = t`Update available: ${update.installed} → ${update.version}`
  return (
    <Tooltip title={text}>
      <Box
        role="img"
        aria-label={text}
        sx={{ display: 'flex', flexShrink: 0, color: 'primary.main' }}
      >
        <ArrowUp size={16} />
      </Box>
    </Tooltip>
  )
}

export function ModSwitch({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const setEnabled = useMods((s) => s.setEnabled)
  return (
    <Switch
      size="small"
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
  const { t } = useLingui()
  const showFiles = useMods((s) => s.showFiles)
  return (
    <Button
      variant="outlined"
      sx={{ whiteSpace: 'nowrap' }}
      onClick={() => {
        showFiles(mod).catch(reportUnexpected)
      }}
    >
      {t`Show files`}
    </Button>
  )
}

export function RemoveButton({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const askRemove = useMods((s) => s.askRemove)
  return (
    <Button
      variant="outlined"
      color="error"
      sx={{ whiteSpace: 'nowrap' }}
      onClick={() => askRemove(mod)}
    >
      {t`Remove`}
    </Button>
  )
}

export function RemoveDialog() {
  const { t } = useLingui()
  const mod = useMods((s) => s.removing)
  const mods = useMods((s) => s.mods)
  const askRemove = useMods((s) => s.askRemove)
  const remove = useMods((s) => s.remove)
  const others = mod ? siblingsOf(mods, mod).map((m) => m.name) : []
  const close = () => askRemove(null)
  return (
    <Dialog open={mod !== null} onClose={close} slotProps={{ paper }}>
      <DialogTitle>{t`Remove ${mod?.name} from this profile?`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {others.length > 0
            ? t`It came in one download with ${others.join(', ')}, and all of them are removed together.`
            : t`Its folder in this profile is deleted.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>{t`Cancel`}</Button>
        <Button
          color="error"
          onClick={() => {
            close()
            if (mod) {
              remove(mod).catch(reportUnexpected)
            }
          }}
        >
          {t`Remove`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
