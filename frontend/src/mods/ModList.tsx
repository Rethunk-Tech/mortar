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
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { useDetail } from './detail.ts'
import { kindLabel, modId, sourceKind } from './lookup.ts'
import { useContextMenuSx } from './menu.ts'
import { heading } from './paper.ts'
import { LetterTile, ModSwitch, ProblemBadge, UpdateBadge } from './parts.tsx'

const SELECTED_ALPHA = 0.14
const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const

const COLUMNS = '46px 26px minmax(0,1fr) 130px 80px 100px'
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

function ModRow({ mod: m, striped, profile }: { mod: Mod; striped: boolean; profile: Profile }) {
  const { t } = useLingui()
  const selectedId = useDetail((s) => s.detailId)
  const show = useDetail((s) => s.show)
  const menuSx = useContextMenuSx(m, sourceKind(profile, m) !== 'smapi')
  return (
    <TableRow
      hover={true}
      selected={modId(m) === selectedId}
      onClick={() => show(m)}
      sx={{
        ...rowSx,
        ...menuSx,
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
          smapi: t`SMAPI`,
          archive: t`Archive`,
          nexus: t`Nexus Mods`,
        })}
      </Cell>
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
