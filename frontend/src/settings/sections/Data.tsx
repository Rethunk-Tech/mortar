import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
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
import { Download, FolderOpen, Trash2, Upload } from 'lucide-react'
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
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  ApplyImportedSettings,
  ExportSettings,
  OpenDataFolder,
  PreviewImportSettings,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
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

function changeLine(field: string, from: string, to: string) {
  switch (field) {
    case 'accent':
      return i18n._(msg`Accent: ${from} → ${to}`)
    case 'background':
      return i18n._(msg`Background: ${from} → ${to}`)
    case 'lastGame':
      return i18n._(msg`Last game: ${from} → ${to}`)
    case 'backupsKept':
      return i18n._(msg`Backups kept: ${from} → ${to}`)
    case 'listColumns':
      return i18n._(msg`List columns: ${from} → ${to}`)
    case 'listSortColumn':
      return i18n._(msg`List sort: ${from} → ${to}`)
    case 'listSortDir':
      return i18n._(msg`List sort direction: ${from} → ${to}`)
    case 'listGroupBy':
      return i18n._(msg`List grouping: ${from} → ${to}`)
    case 'checkModUpdatesOnStart':
      return i18n._(msg`Check mod updates on start: ${from} → ${to}`)
    case 'tellWhenSmapiOut':
      return i18n._(msg`Tell when SMAPI is out: ${from} → ${to}`)
    case 'tipsSeen':
      return i18n._(msg`Seen tips: ${from} → ${to}`)
    default:
      return i18n._(msg`${field}: ${from} → ${to}`)
  }
}

function ImportSettingsDialog({
  preview,
  onClose,
}: {
  preview: ImportPreview | null
  onClose: () => void
}) {
  const { t } = useLingui()
  const changes = preview?.changes ?? []
  return (
    <Dialog open={preview !== null} onClose={onClose} transitionDuration={0} slotProps={{ paper }}>
      <DialogTitle>{t`Import settings`}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 360 }}>
        {changes.length === 0 ? (
          <Box sx={{ fontSize: 13 }}>{t`Nothing would change.`}</Box>
        ) : (
          changes.map((c) => (
            <Box key={c.field} sx={{ fontSize: 13 }}>
              {changeLine(c.field, c.from, c.to)}
            </Box>
          ))
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} sx={nowrap}>
          {t`Cancel`}
        </Button>
        <Button
          onClick={() => {
            if (!preview?.raw) {
              return
            }
            ApplyImportedSettings(preview.raw).then(onClose).catch(reportUnexpected)
          }}
          disabled={changes.length === 0}
          sx={nowrap}
        >
          {t`Import`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export function Data() {
  const { t } = useLingui()
  const [usage, setUsage] = useState<DiskUse | null>(null)
  const [bytes, setBytes] = useState(0)
  const [preview, setPreview] = useState<Preview | null>(null)
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null)
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
                  setImportPreview(next)
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
      <ImportSettingsDialog preview={importPreview} onClose={() => setImportPreview(null)} />
    </Box>
  )
}
