import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableSortLabel,
} from '@mui/material'
import { Download, Upload } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import type {
  ModUsage as ModsDisk,
  ModUse,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  ClearCache,
  ModUsage as LoadModUsage,
  RemoveStoreItem,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  ExportSettings,
  PreviewImportSettings,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { When } from '../../i18n/When.tsx'
import { paper } from '../../mods/paper.ts'
import { formatBytes } from '../../saves/backupFormat.ts'
import { reportUnexpected } from '../../toasts/report.ts'

const nowrap = { whiteSpace: 'nowrap' } as const
const mono = { fontFamily: '"IBM Plex Mono", monospace', fontSize: 13 } as const
const ellipsis = { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' } as const

function Row({ label, size }: { label: string; size: number }) {
  return (
    <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 2, fontSize: 13 }}>
      <Box sx={{ minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {label}
      </Box>
      <Box sx={{ ...mono, flexShrink: 0 }}>{formatBytes(size)}</Box>
    </Box>
  )
}

type ModCol = 'name' | 'size' | 'profiles' | 'profileSize' | 'lastUsed'

function sortModUses(items: readonly ModUse[], col: ModCol, dir: 'asc' | 'desc'): ModUse[] {
  const mul = dir === 'asc' ? 1 : -1
  return [...items].sort((a, b) => {
    let c = 0
    switch (col) {
      case 'name':
        c = a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' })
        break
      case 'size':
        c = a.size - b.size
        break
      case 'profiles':
        c = a.profiles - b.profiles
        break
      case 'profileSize':
        c = a.profileSize - b.profileSize
        break
      case 'lastUsed':
        c = (a.lastUsed || '').localeCompare(b.lastUsed || '')
        break
      default:
        c = 0
    }
    if (c === 0) {
      c = a.key.localeCompare(b.key)
    }
    return c * mul
  })
}

function DataByMod({ onCleanup, onChanged }: { onCleanup: () => void; onChanged: () => void }) {
  const { t } = useLingui()
  const [usage, setUsage] = useState<ModsDisk | null>(null)
  const [col, setCol] = useState<ModCol>('size')
  const [dir, setDir] = useState<'asc' | 'desc'>('desc')
  const [removeItem, setRemoveItem] = useState<ModUse | null>(null)
  useEffect(() => {
    LoadModUsage().then(setUsage).catch(reportUnexpected)
  }, [])
  const click = (next: ModCol) => {
    if (col === next) {
      setDir(dir === 'asc' ? 'desc' : 'asc')
      return
    }
    setCol(next)
    setDir(next === 'name' ? 'asc' : 'desc')
  }
  const items = useMemo(() => sortModUses(usage?.items ?? [], col, dir), [usage, col, dir])
  const head = (id: ModCol, label: string) => (
    <TableCell sx={{ whiteSpace: 'nowrap' }}>
      <TableSortLabel
        active={col === id}
        direction={col === id ? dir : 'asc'}
        onClick={() => click(id)}
      >
        {label}
      </TableSortLabel>
    </TableCell>
  )
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, pt: 1 }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Disk use by mod`}</Box>
      {usage ? <Row label={t`Total`} size={usage.total} /> : <LinearProgress />}
      <Table size="small" sx={{ '& td, & th': { px: 1, py: 0.75, border: 0, fontSize: 13 } }}>
        <TableHead>
          <TableRow>
            {head('name', t`Name`)}
            {head('size', t`Size`)}
            {head('profiles', t`Profiles`)}
            {head('profileSize', t`Profile copies`)}
            {head('lastUsed', t`Last used`)}
            <TableCell sx={nowrap}>{t`Actions`}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {items.map((it) => (
            <TableRow key={`${it.game}/${it.key}`}>
              <TableCell sx={{ ...ellipsis, maxWidth: 220 }} title={it.key}>
                {it.name}
              </TableCell>
              <TableCell sx={mono}>{formatBytes(it.size)}</TableCell>
              <TableCell>{it.profiles}</TableCell>
              <TableCell sx={mono}>{formatBytes(it.profileSize)}</TableCell>
              <TableCell sx={nowrap}>{it.lastUsed ? <When value={it.lastUsed} /> : '—'}</TableCell>
              <TableCell sx={nowrap}>
                {it.profiles === 0 ? (
                  <Button size="small" onClick={() => setRemoveItem(it)} sx={nowrap}>
                    {t`Remove`}
                  </Button>
                ) : null}
                <Button size="small" onClick={onCleanup} sx={nowrap}>
                  {t`Clean up`}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <Dialog
        open={removeItem !== null}
        onClose={() => setRemoveItem(null)}
        transitionDuration={0}
        slotProps={{ paper }}
      >
        <DialogTitle>{t`Remove from the store`}</DialogTitle>
        <DialogContent>
          {removeItem ? t`Remove ${removeItem.name} from the store?` : null}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRemoveItem(null)} sx={nowrap}>{t`Cancel`}</Button>
          <Button
            onClick={() => {
              if (!removeItem) {
                return
              }
              RemoveStoreItem(removeItem.game, removeItem.key)
                .then(() => {
                  setRemoveItem(null)
                  onChanged()
                })
                .catch(reportUnexpected)
            }}
            sx={nowrap}
          >
            {t`Remove`}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

function CacheClearDialog({
  open,
  onClose,
  onCleared,
}: {
  open: boolean
  onClose: () => void
  onCleared: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog open={open} onClose={onClose} transitionDuration={0} slotProps={{ paper }}>
      <DialogTitle>{t`Clear cache`}</DialogTitle>
      <DialogContent>{t`Problem scans rebuild the cache on the next check.`}</DialogContent>
      <DialogActions>
        <Button onClick={onClose} sx={nowrap}>{t`Cancel`}</Button>
        <Button
          onClick={() => {
            ClearCache()
              .then(() => {
                onClose()
                onCleared()
              })
              .catch(reportUnexpected)
          }}
          sx={nowrap}
        >
          {t`Clear`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function DataSettingsFiles({ onImport }: { onImport: (p: ImportPreview) => void }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
      <Button
        onClick={() => ExportSettings().catch(reportUnexpected)}
        startIcon={<Download size={16} />}
        sx={{ alignSelf: 'flex-start', ...nowrap }}
      >
        {t`Export settings…`}
      </Button>
      <Button
        onClick={() => {
          PreviewImportSettings()
            .then((next) => {
              if (next.raw) {
                onImport(next)
              }
            })
            .catch(reportUnexpected)
        }}
        startIcon={<Upload size={16} />}
        sx={{ alignSelf: 'flex-start', ...nowrap }}
      >
        {t`Import settings…`}
      </Button>
    </Box>
  )
}

export { CacheClearDialog, DataByMod, DataSettingsFiles }
