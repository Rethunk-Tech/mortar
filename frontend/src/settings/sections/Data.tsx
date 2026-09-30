import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
  Link,
} from '@mui/material'
import { FolderOpen, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import type {
  Usage as DiskUse,
  Preview,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  Cleanup,
  CleanupPreview,
  Usage,
  UsageProgress,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import { OpenDataFolder } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { paper } from '../../mods/paper.ts'
import { formatBytes } from '../../saves/backupFormat.ts'
import { reportUnexpected } from '../../toasts/report.ts'

const nowrap = { whiteSpace: 'nowrap' } as const
const mono = { fontFamily: '"IBM Plex Mono", monospace', fontSize: 13 } as const
const PROGRESS_MS = 80

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

export function Data() {
  const { t } = useLingui()
  const [usage, setUsage] = useState<DiskUse | null>(null)
  const [bytes, setBytes] = useState(0)
  const [preview, setPreview] = useState<Preview | null>(null)
  const [busy, setBusy] = useState(false)
  const load = useCallback(() => {
    setUsage(null)
    const tick = globalThis.setInterval(() => {
      UsageProgress()
        .then((p) => {
          if (p.measuring) {
            setBytes(p.bytes)
          }
        })
        .catch(() => undefined)
    }, PROGRESS_MS)
    Usage()
      .then(setUsage)
      .catch(reportUnexpected)
      .finally(() => {
        globalThis.clearInterval(tick)
      })
  }, [])
  useEffect(() => {
    load()
  }, [load])
  const openPreview = () => {
    CleanupPreview().then(setPreview).catch(reportUnexpected)
  }
  const runCleanup = () => {
    setBusy(true)
    Cleanup()
      .then(() => {
        setPreview(null)
        load()
      })
      .catch(reportUnexpected)
      .finally(() => setBusy(false))
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
        <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Mortar's data`}</Box>
        <Box sx={{ ...mono, wordBreak: 'break-all' }}>{usage ? usage.path : t`Measuring…`}</Box>
        <Link
          component="button"
          onClick={() => OpenDataFolder().catch(reportUnexpected)}
          sx={{
            alignSelf: 'flex-start',
            fontSize: 13,
            ...nowrap,
            display: 'inline-flex',
            gap: 0.75,
            alignItems: 'center',
          }}
        >
          <FolderOpen size={14} />
          {t`Open folder`}
        </Link>
      </Box>
      {usage ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px', maxWidth: 480 }}>
          {(usage.profiles ?? []).map((p) => (
            <Row key={`${p.game}/${p.id}`} label={t`${p.name} (profile)`} size={p.size} />
          ))}
          <Row label={t`Store`} size={usage.store} />
          <Row label={t`Cache`} size={usage.cache} />
          <Row label={t`Save backups`} size={usage.backups} />
          <Row label={t`Trash`} size={usage.trash} />
          <Row label={t`Total`} size={usage.total} />
        </Box>
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          <LinearProgress />
          <Box sx={{ ...mono, color: 'text.secondary' }}>{formatBytes(bytes)}</Box>
        </Box>
      )}
      <Button
        onClick={openPreview}
        startIcon={<Trash2 size={16} />}
        sx={{ alignSelf: 'flex-start', ...nowrap }}
      >
        {t`Clean up unused`}
      </Button>
      <Dialog
        open={preview !== null}
        onClose={() => setPreview(null)}
        transitionDuration={0}
        slotProps={{ paper }}
      >
        <DialogTitle>{t`Clean up unused`}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 360 }}>
          {(preview?.items ?? []).length === 0 ? (
            <Box sx={{ fontSize: 13 }}>{t`Nothing to remove.`}</Box>
          ) : (
            (preview?.items ?? []).map((it) => <Row key={it.rel} label={it.label} size={it.size} />)
          )}
          {(preview?.items ?? []).length > 0 ? (
            <Row label={t`Total`} size={preview?.total ?? 0} />
          ) : null}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPreview(null)} sx={nowrap}>
            {t`Cancel`}
          </Button>
          <Button
            onClick={runCleanup}
            disabled={busy || (preview?.items ?? []).length === 0}
            sx={nowrap}
          >
            {t`Clean up`}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
