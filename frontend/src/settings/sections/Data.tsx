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
  TextField,
} from '@mui/material'
import { Download, FolderInput, FolderOpen, Trash2, Upload } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type {
  Usage as DiskUse,
  MoveEstimate,
  Preview,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  Cleanup,
  CleanupPreview,
  MoveDataFolder,
  MoveDataFolderPreview,
  Usage,
  UsageProgress,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import { State } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { PickFolder } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  ApplyImportedSettings,
  ExportSettings,
  OpenDataFolder,
  PreviewImportSettings,
  SetBackupsKept,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useLaunch } from '../../launch/store.ts'
import { paper } from '../../mods/paper.ts'
import { useNav } from '../../nav/store.ts'
import { formatBytes } from '../../saves/backupFormat.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { useSettings } from '../store.ts'
import { beginUsageLoad } from '../usageLoad.ts'

const MIN_KEPT = 1
const MAX_KEPT = 50
const MOVE_PROGRESS_INTERVAL = 200
const nowrap = { whiteSpace: 'nowrap' } as const
const mono = { fontFamily: '"IBM Plex Mono", monospace', fontSize: 13 } as const

function BackupsKept() {
  const { t } = useLingui()
  const kept = useSettings((s) => s.backupsKept)
  const push = useToasts((s) => s.push)
  const [draft, setDraft] = useState(String(kept))
  useEffect(() => setDraft(String(kept)), [kept])
  const commit = () => {
    const n = Number(draft)
    if (!Number.isInteger(n) || n < MIN_KEPT || n > MAX_KEPT) {
      setDraft(String(kept))
      return
    }
    if (n !== kept) {
      SetBackupsKept(n).catch((err: unknown) => {
        const body = errorText(err)
        push({ kind: 'error', title: t`Couldn't save that setting`, ...(body ? { body } : {}) })
        setDraft(String(kept))
      })
    }
  }
  return (
    <TextField
      type="number"
      size="small"
      label={t`Backups kept`}
      value={draft}
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === 'Enter' && e.target instanceof HTMLInputElement) {
          e.target.blur()
        }
      }}
      helperText={t`Saves are zipped before mods update; older backups beyond this many are deleted. ${MIN_KEPT} to ${MAX_KEPT}.`}
      slotProps={{ htmlInput: { min: MIN_KEPT, max: MAX_KEPT, step: 1 } }}
      sx={{ alignSelf: 'flex-start', width: 320 }}
    />
  )
}

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
    case 'includePrereleaseModVersions':
      return i18n._(msg`Include pre-release mod versions: ${from} → ${to}`)
    case 'checkOnlyEnabledMods':
      return i18n._(msg`Check only enabled mods: ${from} → ${to}`)
    case 'enableModsWhenInstalled':
      return i18n._(msg`Enable mods when installed: ${from} → ${to}`)
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

function MoveDataButton({ onPicked }: { onPicked: (dest: string) => void }) {
  const { t } = useLingui()
  return (
    <Button
      variant="outlined"
      startIcon={<FolderInput size={16} />}
      onClick={() => {
        const st = useLaunch.getState().status
        if (st?.state === State.Launching || st?.state === State.Running) {
          useToasts.getState().push({
            kind: 'error',
            title: t`Stop the game before moving the data folder.`,
          })
          return
        }
        PickFolder(t`Move data folder…`)
          .then((dest) => (dest ? onPicked(dest) : undefined))
          .catch((err: unknown) => {
            const body = errorText(err)
            useToasts.getState().push({
              kind: 'error',
              title: t`Couldn't prepare the data folder move`,
              ...(body ? { body } : {}),
            })
          })
      }}
      sx={{ ...nowrap, flexShrink: 0 }}
    >
      {t`Move…`}
    </Button>
  )
}

interface MoveState {
  dest: string
  estimate: MoveEstimate
}

function MoveDialog({
  move,
  moving,
  progress,
  error,
  onClose,
  onMove,
}: {
  move: MoveState | null
  moving: boolean
  progress: { files: number; totalFiles: number; bytes: number; totalBytes: number }
  error: string
  onClose: () => void
  onMove: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog
      open={move !== null}
      onClose={() => !moving && onClose()}
      transitionDuration={0}
      slotProps={{ paper }}
    >
      <DialogTitle>{t`Move data folder`}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1, minWidth: 360 }}>
        {move ? (
          <>
            <Box>{t`Data to copy: ${formatBytes(move.estimate.bytes)}`}</Box>
            <Box>{t`Free space at destination: ${formatBytes(move.estimate.freeBytes)}`}</Box>
            {moving ? (
              <>
                <LinearProgress />
                <Box sx={{ ...mono }}>
                  {t`${progress.files}/${progress.totalFiles} files · ${formatBytes(progress.bytes)} / ${formatBytes(progress.totalBytes)}`}
                </Box>
              </>
            ) : null}
            {error ? <Box sx={{ color: 'error.main' }}>{error}</Box> : null}
          </>
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={moving} sx={nowrap}>{t`Cancel`}</Button>
        <Button onClick={onMove} disabled={moving || move === null} sx={nowrap}>{t`Move`}</Button>
      </DialogActions>
    </Dialog>
  )
}

export function Data() {
  const { t } = useLingui()
  const openProfiles = useNav((s) => s.openProfiles)
  const [usage, setUsage] = useState<DiskUse | null>(null)
  const [bytes, setBytes] = useState(0)
  const [preview, setPreview] = useState<Preview | null>(null)
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null)
  const [busy, setBusy] = useState(false)
  const [move, setMove] = useState<MoveState | null>(null)
  const [moveError, setMoveError] = useState('')
  const [moving, setMoving] = useState(false)
  const [moveProgress, setMoveProgress] = useState({
    files: 0,
    totalFiles: 0,
    bytes: 0,
    totalBytes: 0,
  })
  const stopRef = useRef<() => void>(() => {
    return
  })
  const restart = useCallback(() => {
    stopRef.current()
    setUsage(null)
    stopRef.current = beginUsageLoad({
      usage: Usage,
      progress: UsageProgress,
      setBytes,
      setUsage,
      onError: reportUnexpected,
    })
  }, [])
  useEffect(() => {
    restart()
    return () => {
      stopRef.current()
    }
  }, [restart])
  const openPreview = () => {
    CleanupPreview().then(setPreview).catch(reportUnexpected)
  }
  const runCleanup = () => {
    setBusy(true)
    if (!preview) {
      return
    }
    Cleanup(preview)
      .then(() => {
        setPreview(null)
        restart()
      })
      .catch(reportUnexpected)
      .finally(() => setBusy(false))
  }
  const prepareMove = (dest: string) => {
    MoveDataFolderPreview(dest)
      .then((estimate) => {
        setMoveError('')
        setMove({ dest, estimate })
      })
      .catch((err: unknown) => {
        const body = errorText(err)
        useToasts.getState().push({
          kind: 'error',
          title: t`Couldn't inspect the destination folder`,
          ...(body ? { body } : {}),
        })
      })
  }
  const runMove = () => {
    if (!move) {
      return
    }
    setMoving(true)
    setMoveError('')
    const poll = globalThis.setInterval(() => {
      UsageProgress().then(
        (progress) =>
          setMoveProgress({
            files: progress.files,
            totalFiles: progress.totalFiles,
            bytes: progress.bytes,
            totalBytes: progress.totalBytes,
          }),
        () => undefined,
      )
    }, MOVE_PROGRESS_INTERVAL)
    MoveDataFolder(move.dest)
      .then(() => setMove(null))
      .catch((err: unknown) => setMoveError(errorText(err) || t`Could not move the data folder.`))
      .finally(() => {
        globalThis.clearInterval(poll)
        setMoving(false)
      })
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          p: '14px',
          bgcolor: 'rgba(55,55,65,0.9)',
          borderRadius: '6px',
        }}
      >
        <Box
          sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: '4px' }}
        >
          <Box sx={{ fontSize: 15, fontWeight: 600 }}>{t`Mortar's data`}</Box>
          <Box sx={{ ...mono, wordBreak: 'break-all' }}>{usage ? usage.path : t`Measuring…`}</Box>
        </Box>
        <Button
          variant="outlined"
          startIcon={<FolderOpen size={16} />}
          onClick={() => OpenDataFolder().catch(reportUnexpected)}
          sx={{ ...nowrap, flexShrink: 0 }}
        >
          {t`Open folder`}
        </Button>
        <MoveDataButton onPicked={prepareMove} />
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
          <Box
            sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2 }}
          >
            <Row label={t`Trash`} size={usage.trash} />
            <Button size="small" onClick={openProfiles} sx={{ ...nowrap, flexShrink: 0 }}>
              {t`Manage deleted profiles`}
            </Button>
          </Box>
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
      <Box sx={{ fontSize: 14, fontWeight: 600, pt: 1 }}>{t`Save backups`}</Box>
      <BackupsKept />
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
      <MoveDialog
        move={move}
        moving={moving}
        progress={moveProgress}
        error={moveError}
        onClose={() => setMove(null)}
        onMove={runMove}
      />
    </Box>
  )
}
