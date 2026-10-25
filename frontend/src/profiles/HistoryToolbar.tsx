import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { GitCompare, RotateCcw, ShieldCheck } from 'lucide-react'
import { DialogFooter } from '../shell/dialog/AppDialog.tsx'

/** History's footer: what the reader can do with the timeline, and Close. */
export function HistoryToolbar({
  busy,
  canRestore,
  comparing,
  picked,
  onMark,
  onRestore,
  onToggleCompare,
  onCompare,
  onClose,
}: {
  busy: boolean
  // False until a point in the history is marked known good.
  canRestore: boolean
  comparing: boolean
  picked: number
  onMark: () => void
  onRestore: () => void
  onToggleCompare: () => void
  onCompare: () => void
  onClose: () => void
}) {
  const { t } = useLingui()
  let aside = ''
  if (comparing) {
    aside = t`Pick two entries (${picked} of 2)`
  } else if (!canRestore) {
    aside = t`Mark a known good point first.`
  }
  return (
    <DialogFooter aside={aside}>
      <Button onClick={onClose}>{t`Close`}</Button>
      <Button
        startIcon={<GitCompare size={16} />}
        aria-pressed={comparing}
        disabled={busy}
        onClick={onToggleCompare}
      >
        {comparing ? t`Stop comparing` : t`Compare two…`}
      </Button>
      {comparing ? (
        <Button variant="contained" disabled={picked !== 2} onClick={onCompare}>
          {t`Compare`}
        </Button>
      ) : (
        <>
          <Button
            startIcon={<RotateCcw size={16} />}
            disabled={busy || !canRestore}
            onClick={onRestore}
          >
            {t`Restore known good`}
          </Button>
          <Button startIcon={<ShieldCheck size={16} />} disabled={busy} onClick={onMark}>
            {t`Mark current as known good`}
          </Button>
        </>
      )}
    </DialogFooter>
  )
}
