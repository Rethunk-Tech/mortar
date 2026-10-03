import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Switch, Typography } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { useLocked } from './useLocked.ts'

function ModsGroupHeader({
  label,
  count,
  open,
  onToggle,
  hint,
  enabled,
  onEnabled,
}: {
  label: string
  count: number
  open: boolean
  onToggle: () => void
  hint?: string
  enabled?: boolean
  onEnabled?: (on: boolean) => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  return (
    <ButtonBase
      onClick={onToggle}
      aria-expanded={open}
      title={hint}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.75,
        width: '100%',
        px: 2,
        py: 0.75,
        justifyContent: 'flex-start',
        textAlign: 'left',
        fontFamily: 'inherit',
        color: 'inherit',
        bgcolor: 'var(--mortar-hairline-ghost)',
        borderBottom: '1px solid var(--mortar-hairline-muted)',
      }}
    >
      {open ? (
        <ChevronDown size={14} aria-hidden={true} />
      ) : (
        <ChevronRight size={14} aria-hidden={true} />
      )}
      <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{label}</Typography>
      <Box component="span" sx={{ fontSize: 12, color: 'text.secondary' }}>
        {count}
      </Box>
      {onEnabled ? (
        <Box
          sx={{ ml: 'auto' }}
          onClick={(e) => e.stopPropagation()}
          onKeyDown={(e) => e.stopPropagation()}
        >
          <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
            <Switch
              size="small"
              checked={enabled === true}
              disabled={locked}
              onChange={(_, on) => onEnabled(on)}
              slotProps={{ input: { 'aria-label': label } }}
            />
          </DisabledReason>
        </Box>
      ) : null}
    </ButtonBase>
  )
}

export { ModsGroupHeader }
