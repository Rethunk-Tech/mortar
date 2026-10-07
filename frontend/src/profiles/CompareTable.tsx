import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Checkbox, Typography } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import type { CompareGroup, CompareRow, CompareSide } from './compare.ts'
import { compareOpen } from './compareOpen.ts'

const ARROW = '→'

function openRow(row: CompareRow, profileA: Profile, profileB: Profile) {
  if (row.a) {
    return compareOpen(profileA, row.a)
  }
  return row.b ? compareOpen(profileB, row.b) : undefined
}

const COLUMNS = '22px minmax(0,1fr) 110px 24px 110px'

export function CompareHeader({ aName, bName }: { aName: string; bName: string }) {
  const { t } = useLingui()
  const cell = {
    fontSize: 11,
    fontWeight: 700,
    letterSpacing: '0.08em',
    color: 'text.secondary',
    textTransform: 'uppercase',
  } as const
  return (
    <Box sx={{ display: 'grid', gridTemplateColumns: COLUMNS, gap: 1.25, px: 0.5, pb: 0.75 }}>
      <span />
      <Typography sx={cell}>{t`Mod`}</Typography>
      <Typography noWrap={true} sx={{ ...cell, textAlign: 'right' }}>
        {aName}
      </Typography>
      <span />
      <Typography noWrap={true} sx={cell}>
        {bName}
      </Typography>
    </Box>
  )
}

export function CompareGroupView({
  group,
  title,
  profileA,
  profileB,
  isChecked,
  onToggle,
  disabled,
}: {
  group: CompareGroup
  title: string
  profileA: Profile
  profileB: Profile
  isChecked: (row: CompareRow) => boolean
  onToggle: (row: CompareRow) => void
  disabled: boolean
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(true)
  const Chevron = open ? ChevronDown : ChevronRight
  const cellText = (row: CompareRow, side: CompareSide | null) => {
    if (!side) {
      return '·'
    }
    if (row.kind === 'enabled') {
      return side.enabled ? t`Enabled` : t`Disabled`
    }
    return row.kind === 'source' ? sourceLabel(side.source.kind) : side.version
  }
  return (
    <Box>
      <ButtonBase
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          height: 36,
          width: 1,
          justifyContent: 'flex-start',
          font: 'inherit',
          fontWeight: 600,
        }}
      >
        <Chevron size={14} aria-hidden={true} />
        {title}
        <Typography component="span" color="text.secondary" sx={{ fontWeight: 400 }}>
          {group.rows.length}
        </Typography>
      </ButtonBase>
      {open
        ? group.rows.map((row) => {
            const opener = openRow(row, profileA, profileB)
            return (
              <Box
                key={row.id}
                sx={{
                  display: 'grid',
                  gridTemplateColumns: COLUMNS,
                  gap: 1.25,
                  alignItems: 'center',
                  height: 38,
                  px: 0.5,
                  borderTop: 1,
                  borderColor: 'divider',
                }}
              >
                {row.kind === 'identical' ? (
                  <span />
                ) : (
                  <Checkbox
                    size="small"
                    sx={{ p: 0 }}
                    checked={isChecked(row)}
                    disabled={disabled}
                    onChange={() => onToggle(row)}
                    slotProps={{ input: { 'aria-label': row.name } }}
                  />
                )}
                <Typography
                  noWrap={true}
                  title={row.name}
                  sx={{
                    fontSize: 14,
                    cursor: 'pointer',
                    '&:hover': { textDecoration: 'underline' },
                  }}
                  {...opener}
                >
                  {row.name}
                </Typography>
                <Typography
                  noWrap={true}
                  color="text.secondary"
                  sx={{ textAlign: 'right', fontSize: 14 }}
                >
                  {cellText(row, row.a)}
                </Typography>
                <Typography color="text.secondary" sx={{ textAlign: 'center' }}>
                  {ARROW}
                </Typography>
                <Typography noWrap={true} sx={{ fontSize: 14 }}>
                  {cellText(row, row.b)}
                </Typography>
              </Box>
            )
          })
        : null}
    </Box>
  )
}
