import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, TextField } from '@mui/material'
import { FolderOpen } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Usage as DiskUse } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/models.ts'
import {
  OpenDataFolder,
  SetBackupsKept,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { useSettings } from '../store.ts'
import { MoveDataButton } from './DataMove.tsx'
import { Row } from './DataUi.tsx'
import { mono, nowrap } from './dataStyles.ts'

const MIN_KEPT = 1
const MAX_KEPT = 50

export function BackupsKept() {
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

export function UsageSummary({
  usage,
  bytes,
  openProfiles,
  onClearCache,
}: {
  usage: DiskUse | null
  bytes: number
  openProfiles: () => void
  onClearCache: () => void
}) {
  const { t } = useLingui()
  return usage ? (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px', maxWidth: 480 }}>
      {(usage.profiles ?? []).map((p) => (
        <Row key={`${p.game}/${p.id}`} label={t`${p.name} (profile)`} size={p.size} />
      ))}
      <Row label={t`Store`} size={usage.store} />
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2 }}>
        <Row label={t`Cache`} size={usage.cache} />
        <Button size="small" onClick={onClearCache} sx={{ ...nowrap, flexShrink: 0 }}>
          {t`Clear`}
        </Button>
      </Box>
      <Row label={t`Save backups`} size={usage.backups} />
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2 }}>
        <Row label={t`Trash`} size={usage.trash} />
        <Button size="small" onClick={openProfiles} sx={{ ...nowrap, flexShrink: 0 }}>
          {t`Manage deleted profiles`}
        </Button>
      </Box>
      {usage.sharedSavedKnown ? (
        <Row label={t`Space saved by sharing files`} size={usage.sharedSaved} />
      ) : null}
      <Row label={t`Total`} size={usage.total} />
    </Box>
  ) : (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <LinearProgress />
      <Box sx={{ ...mono, color: 'text.secondary' }}>{formatBytes(bytes)}</Box>
    </Box>
  )
}

export function DataFolderCard({
  usage,
  onPicked,
}: {
  usage: DiskUse | null
  onPicked: (dest: string) => void
}) {
  const { t } = useLingui()
  return (
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
      <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: '4px' }}>
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
      <MoveDataButton onPicked={onPicked} />
    </Box>
  )
}
