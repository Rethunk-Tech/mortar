import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Chip, IconButton, Switch, Tooltip } from '@mui/material'
import { CornerDownRight, Trash2 } from 'lucide-react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { LockedReason } from './LockedReason.tsx'
import { removeOverlay, setOverlayEnabled } from './overlayActions.ts'
import { overlaysByBase } from './overlayRows.ts'
import { useLocked } from './useLocked.ts'
import { LIST_ROW_PX, type OverlayRow } from './virtualRows.ts'

const chipSx = {
  height: 20,
  fontSize: 11,
  bgcolor: 'action.selected',
  color: 'text.secondary',
  '& .MuiChip-label': { px: 0.75 },
} as const

/** An optional file under its main file's row: its own switch and Remove. */
export function OverlayListRow({ overlay }: { overlay: OverlayRow }) {
  const { t } = useLingui()
  const locked = useLocked()
  const dim = !(overlay.enabled && overlay.baseEnabled)
  return (
    <Box
      role="row"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        height: LIST_ROW_PX,
        pl: 5,
        pr: 2,
        fontSize: 13,
        color: dim ? 'text.secondary' : 'text.primary',
      }}
    >
      <Box component={CornerDownRight} size={14} aria-hidden={true} sx={{ flexShrink: 0 }} />
      <Box role="cell" sx={{ display: 'flex' }}>
        <LockedReason locked={locked}>
          <Switch
            size="small"
            checked={overlay.enabled}
            disabled={locked}
            onChange={(e) => {
              setOverlayEnabled(overlay, e.target.checked).catch(reportUnexpected)
            }}
            slotProps={{
              input: {
                'aria-label': overlay.enabled
                  ? t`Switch off ${overlay.label}`
                  : t`Switch on ${overlay.label}`,
              },
            }}
          />
        </LockedReason>
      </Box>
      <Box
        role="cell"
        title={overlay.label}
        sx={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}
      >
        {overlay.label}
      </Box>
      <Tooltip
        title={
          overlay.baseEnabled
            ? t`Replaces some files of the mod above`
            : t`Off while the mod above is switched off`
        }
      >
        <Chip size="small" label={t`Optional file`} sx={chipSx} />
      </Tooltip>
      <Box role="cell" sx={{ ml: 'auto', display: 'flex' }}>
        <LockedReason locked={locked}>
          <Tooltip title={t`Remove ${overlay.label}`}>
            <span>
              <IconButton
                size="small"
                disabled={locked}
                aria-label={t`Remove ${overlay.label}`}
                onClick={() => {
                  removeOverlay(overlay).catch(reportUnexpected)
                }}
              >
                <Trash2 size={14} />
              </IconButton>
            </span>
          </Tooltip>
        </LockedReason>
      </Box>
    </Box>
  )
}

/** "N optional files" on a main file's row when optional files are laid over it. */
export function OverlayCountChip({ mod, profile }: { mod: Mod; profile: Profile }) {
  const { i18n } = useLingui()
  const overlays = overlaysByBase(profile).get(mod.key) ?? []
  if (overlays.length === 0) {
    return null
  }
  return (
    <Tooltip title={overlays.map((o) => o.label).join('\n')}>
      <Chip
        size="small"
        label={i18n._(
          plural(overlays.length, { one: '# optional file', other: '# optional files' }),
        )}
        sx={chipSx}
      />
    </Tooltip>
  )
}
