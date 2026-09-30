import { useLingui } from '@lingui/react/macro'
import {
  alpha,
  Box,
  Drawer,
  Table,
  TableBody,
  TableCell,
  type TableCellProps,
  TableHead,
  TableRow,
  Typography,
  useMediaQuery,
} from '@mui/material'
import { useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact, compactQuery } from '../game/compact.ts'
import { siblingsOf, sourceKind } from './lookup.ts'
import { LetterTile, ModSwitch, RemoveButton, ShowFilesButton } from './parts.tsx'
import { useMods } from './store.ts'

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

const heading = {
  fontSize: 12,
  fontWeight: 700,
  letterSpacing: '0.06em',
  textTransform: 'uppercase',
  color: 'text.secondary',
} as const

const cellBase = { p: 0, border: 0, fontSize: 'inherit', color: 'inherit' } as const

function Cell({ sx, ...props }: TableCellProps) {
  return <TableCell {...props} sx={{ ...cellBase, ...sx }} />
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography sx={heading}>{label}</Typography>
      <Typography sx={{ fontSize: 13, overflowWrap: 'anywhere' }}>{value}</Typography>
    </Box>
  )
}

function Inspector({ mod, source }: { mod: Mod; source: string }) {
  const { t } = useLingui()
  const all = useMods((s) => s.mods)
  const others = siblingsOf(all, mod)
  return (
    <Box sx={{ p: 1.75, display: 'flex', flexDirection: 'column', gap: 1.25, minHeight: '100%' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <LetterTile mod={mod} size={52} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography sx={{ fontSize: 16, fontWeight: 700, overflowWrap: 'anywhere' }}>
            {mod.name}
          </Typography>
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {`${mod.author} · ${source}`}
          </Typography>
        </Box>
        <ModSwitch mod={mod} />
      </Box>
      <Field label={t`Version`} value={mod.version} />
      <Field label={t`UniqueID`} value={mod.uniqueId} />
      {others.length > 0 ? (
        <Box>
          <Typography sx={heading}>{t`In the same download`}</Typography>
          {others.map((o) => (
            <Typography key={o.uniqueId} sx={{ fontSize: 13 }}>
              {o.name}
            </Typography>
          ))}
        </Box>
      ) : null}
      <Box sx={{ flexGrow: 1 }} />
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 1 }}>
        <ShowFilesButton mod={mod} />
        <RemoveButton mod={mod} />
      </Box>
    </Box>
  )
}

export function ModList({ profile, mods }: { profile: Profile; mods: Mod[] }) {
  const { t } = useLingui()
  const narrow = useMediaQuery(compactQuery)
  const [selectedId, setSelectedId] = useState('')
  const selected = mods.find((m) => m.uniqueId === selectedId)
  const kindLabel = (kind: string) => {
    if (kind === 'smapi') {
      return t`SMAPI`
    }
    return kind === 'local' ? t`Archive` : kind
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'grid', gridTemplateColumns: 'minmax(0,1fr) auto' }}>
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
              <TableRow
                key={m.uniqueId}
                hover={true}
                selected={m.uniqueId === selected?.uniqueId}
                onClick={() => setSelectedId(m.uniqueId)}
                sx={{
                  ...rowSx,
                  height: 36,
                  fontSize: 14,
                  cursor: 'pointer',
                  bgcolor: (th) => {
                    if (m.uniqueId === selected?.uniqueId) {
                      return alpha(th.palette.primary.main, SELECTED_ALPHA)
                    }
                    return i % 2 ? 'rgba(255,255,255,0.03)' : 'transparent'
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
                <Cell sx={{ ...ellipsis, color: 'text.secondary', ...hideCompact }}>
                  {m.author}
                </Cell>
                <Cell sx={{ color: 'text.secondary', ...hideCompact }}>
                  {kindLabel(sourceKind(profile, m))}
                </Cell>
                <Cell
                  sx={{
                    fontSize: 13,
                    whiteSpace: 'nowrap',
                    color: m.enabled ? 'text.primary' : 'text.secondary',
                  }}
                >
                  {m.enabled ? t`Enabled` : t`Off`}
                </Cell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Box>
      {narrow ? (
        <Drawer
          anchor="right"
          open={selected !== undefined}
          onClose={() => setSelectedId('')}
          sx={{ top: 'var(--title-bar)' }}
          slotProps={{
            paper: {
              sx: {
                width: 320,
                top: 'var(--title-bar)',
                height: 'calc(100% - var(--title-bar))',
                bgcolor: 'rgba(40,40,48,0.92)',
              },
            },
          }}
        >
          {selected ? (
            <Inspector mod={selected} source={kindLabel(sourceKind(profile, selected))} />
          ) : null}
        </Drawer>
      ) : (
        <Box
          aria-label={t`Selected mod`}
          component="aside"
          sx={{
            width: 300,
            overflowY: 'auto',
            bgcolor: 'rgba(40,40,48,0.72)',
            borderLeft: '1px solid rgba(255,255,255,0.1)',
          }}
        >
          {selected ? (
            <Inspector mod={selected} source={kindLabel(sourceKind(profile, selected))} />
          ) : (
            <Typography sx={{ p: 1.75, fontSize: 13, color: 'text.secondary' }}>
              {t`Select a mod to see its details.`}
            </Typography>
          )}
        </Box>
      )}
    </Box>
  )
}
