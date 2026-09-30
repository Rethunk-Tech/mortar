import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Drawer,
  Table,
  TableBody,
  TableCell,
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
import { compactQuery } from '../game/compact.ts'
import { siblingsOf, sourceKind } from './lookup.ts'
import { LetterTile, ModSwitch, RemoveButton, ShowFilesButton } from './parts.tsx'
import { useMods } from './store.ts'

function Field({ label, value }: { label: string; value: string }) {
  return (
    <Box>
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{label}</Typography>
      <Typography sx={{ overflowWrap: 'anywhere' }}>{value}</Typography>
    </Box>
  )
}

function Inspector({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const all = useMods((s) => s.mods)
  const others = siblingsOf(all, mod)
  return (
    <Box sx={{ p: 2, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <LetterTile mod={mod} size={48} />
        <Typography sx={{ flex: 1, fontSize: 20, fontWeight: 600, overflowWrap: 'anywhere' }}>
          {mod.name}
        </Typography>
        <ModSwitch mod={mod} />
      </Box>
      <Field label={t`Author`} value={mod.author} />
      <Field label={t`Version`} value={mod.version} />
      <Field label={t`UniqueID`} value={mod.uniqueId} />
      {others.length > 0 ? (
        <Box>
          <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
            {t`In the same download`}
          </Typography>
          {others.map((o) => (
            <Typography key={o.uniqueId}>{o.name}</Typography>
          ))}
        </Box>
      ) : null}
      <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
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
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', gap: 2 }}>
      <Box sx={{ flex: 1, minWidth: 0, overflowY: 'auto' }}>
        <Table size="small" stickyHeader={true}>
          <TableHead>
            <TableRow>
              <TableCell padding="checkbox" />
              <TableCell padding="checkbox" />
              <TableCell>{t`Name`}</TableCell>
              <TableCell>{t`Author`}</TableCell>
              <TableCell>{t`Source`}</TableCell>
              <TableCell>{t`Status`}</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {mods.map((m) => (
              <TableRow
                key={m.uniqueId}
                hover={true}
                selected={m.uniqueId === selected?.uniqueId}
                onClick={() => setSelectedId(m.uniqueId)}
                sx={{ cursor: 'pointer' }}
              >
                <TableCell sx={{ width: 72, py: 0 }}>
                  <ModSwitch mod={m} />
                </TableCell>
                <TableCell padding="checkbox">
                  <LetterTile mod={m} size={28} />
                </TableCell>
                <TableCell>{m.name}</TableCell>
                <TableCell>{m.author}</TableCell>
                <TableCell>{kindLabel(sourceKind(profile, m))}</TableCell>
                <TableCell>{m.enabled ? t`Enabled` : t`Disabled`}</TableCell>
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
          sx={{ top: 36 }}
          slotProps={{
            paper: {
              sx: {
                width: 320,
                top: 36,
                height: 'calc(100% - 36px)',
                bgcolor: 'rgba(40,40,48,0.92)',
              },
            },
          }}
        >
          {selected ? <Inspector mod={selected} /> : null}
        </Drawer>
      ) : (
        <Box
          sx={{
            width: 320,
            flexShrink: 0,
            overflowY: 'auto',
            bgcolor: 'background.paper',
            borderRadius: '6px',
          }}
        >
          {selected ? (
            <Inspector mod={selected} />
          ) : (
            <Typography sx={{ p: 2, color: 'text.secondary' }}>
              {t`Select a mod to see its details.`}
            </Typography>
          )}
        </Box>
      )}
    </Box>
  )
}
