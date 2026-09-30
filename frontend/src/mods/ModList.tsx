import { useLingui } from '@lingui/react/macro'
import {
  alpha,
  Box,
  Table,
  TableBody,
  TableCell,
  type TableCellProps,
  TableHead,
  TableRow,
} from '@mui/material'
import { useEffect } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDetail } from './detail.ts'
import { kindLabel, modId, nexusIdOf, sourceKind } from './lookup.ts'
import { contextMenuProps } from './menu.ts'
import { primeDetails, useNexusDetails } from './nexusDetails.ts'
import { heading } from './paper.ts'
import { LetterTile, ModSwitch, ProblemBadge, UpdateBadge } from './parts.tsx'

const SELECTED_ALPHA = 0.14
const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const

const COLUMNS = '46px 26px minmax(0,1fr) 130px 80px 130px 100px'
const rowSx = {
  display: 'grid',
  gridTemplateColumns: COLUMNS,
  gap: '10px',
  alignItems: 'center',
  px: 2,
  [compact]: { gridTemplateColumns: '46px 26px minmax(0,1fr) 80px' },
}
const hideCompact = { [compact]: { display: 'none' } }

const cellBase = { p: 0, border: 0, fontSize: 'inherit', color: 'inherit' } as const

function Cell({ sx, ...props }: TableCellProps) {
  return <TableCell {...props} sx={{ ...cellBase, ...sx }} />
}

function CategoryCell({ nexusId }: { nexusId: number }) {
  const category = useNexusDetails((s) => s.byId[nexusId]?.details?.category) ?? ''
  return (
    <Cell title={category} sx={{ ...ellipsis, color: 'text.secondary', ...hideCompact }}>
      {category || '—'}
    </Cell>
  )
}

function ModRow({ mod: m, striped, profile }: { mod: Mod; striped: boolean; profile: Profile }) {
  const { t } = useLingui()
  const selectedId = useDetail((s) => s.detailId)
  const show = useDetail((s) => s.show)
  return (
    <TableRow
      hover={true}
      selected={modId(m) === selectedId}
      onClick={() => show(m)}
      tabIndex={0}
      {...contextMenuProps(m)}
      sx={{
        ...rowSx,
        height: 36,
        fontSize: 14,
        cursor: 'pointer',
        bgcolor: (th) => {
          if (modId(m) === selectedId) {
            return alpha(th.palette.primary.main, SELECTED_ALPHA)
          }
          return striped ? 'rgba(255,255,255,0.03)' : 'transparent'
        },
      }}
    >
      <Cell>
        <ModSwitch mod={m} />
      </Cell>
      <Cell>
        <LetterTile mod={m} size={26} />
      </Cell>
      <Cell sx={{ ...ellipsis, fontWeight: 500 }}>{m.name}</Cell>
      <Cell sx={{ ...ellipsis, color: 'text.secondary', ...hideCompact }}>{m.author}</Cell>
      <Cell sx={{ color: 'text.secondary', ...hideCompact }}>
        {kindLabel(sourceKind(profile, m), {
          archive: t`Archive`,
          nexus: t`Nexus Mods`,
          github: t`GitHub`,
        })}
      </Cell>
      <CategoryCell nexusId={nexusIdOf(profile, m)} />
      <Cell
        sx={{
          fontSize: 13,
          whiteSpace: 'nowrap',
          color: m.enabled ? 'text.primary' : 'text.secondary',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
          <ProblemBadge mod={m} />
          <UpdateBadge mod={m} />
          {m.enabled ? t`Enabled` : t`Off`}
        </Box>
      </Cell>
    </TableRow>
  )
}

export function ModList({ profile, mods }: { profile: Profile; mods: Mod[] }) {
  const { t } = useLingui()
  useEffect(() => {
    primeDetails(mods.map((m) => nexusIdOf(profile, m)).filter((id) => id > 0)).catch(
      reportUnexpected,
    )
  }, [mods, profile])
  return (
    <Box sx={{ minWidth: 0, minHeight: 0, overflowY: 'auto' }}>
      <Table
        aria-label={t`Mods`}
        stickyHeader={true}
        sx={{
          display: 'block',
          '& thead, & tbody': { display: 'block' },
        }}
      >
        <TableHead>
          <TableRow
            sx={{
              ...rowSx,
              height: 30,
              position: 'sticky',
              top: 0,
              zIndex: 1,
              bgcolor: 'rgba(25,25,30,0.9)',
              ...heading,
              borderBottom: '1px solid rgba(255,255,255,0.08)',
            }}
          >
            <Cell>{t`On`}</Cell>
            <Cell />
            <Cell>{t`Name`}</Cell>
            <Cell sx={hideCompact}>{t`Author`}</Cell>
            <Cell sx={hideCompact}>{t`Source`}</Cell>
            <Cell sx={hideCompact}>{t`Category`}</Cell>
            <Cell>{t`Status`}</Cell>
          </TableRow>
        </TableHead>
        <TableBody>
          {mods.map((m, i) => (
            <ModRow key={modId(m)} mod={m} striped={i % 2 === 1} profile={profile} />
          ))}
        </TableBody>
      </Table>
    </Box>
  )
}
