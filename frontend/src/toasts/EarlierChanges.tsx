import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Collapse, Typography } from '@mui/material'
import { useState } from 'react'
import type { HistoryEvent } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { historyChangeSummary } from '../profiles/historyCounts.ts'
import type { useHistoryPanel } from '../profiles/useHistoryPanel.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { laterEvents } from './history.ts'

type Panel = ReturnType<typeof useHistoryPanel>

function EarlierRow({
  ev,
  panel,
  onUndo,
}: {
  ev: HistoryEvent
  panel: Panel
  onUndo: (ev: HistoryEvent) => void
}) {
  const { t } = useLingui()
  const [diff, setDiff] = useState(false)
  const label = ev.kind === 'good' ? t`Known good` : ev.label
  const changes = historyChangeSummary(ev)
  const details = panel.items[ev.id] ?? []
  const trimmed = ev.kind === 'trimmed'
  return (
    <Box sx={{ px: 1.5, py: 1, borderBottom: '1px solid var(--mortar-hairline-faint)' }}>
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 1 }}>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{label}</Typography>
          <Typography sx={{ fontSize: 11, color: 'text.secondary' }}>
            {changes === '' ? null : `${changes} · `}
            <When value={ev.at} withTime={true} />
          </Typography>
        </Box>
        {details.length > 0 ? (
          <Button size="small" aria-expanded={diff} onClick={() => setDiff(!diff)}>
            {t`Show diff`}
          </Button>
        ) : null}
        {trimmed ? null : (
          <Button size="small" disabled={panel.busy !== ''} onClick={() => onUndo(ev)}>
            {t`Undo`}
          </Button>
        )}
      </Box>
      <Collapse in={diff} unmountOnExit={true}>
        <Box component="ul" sx={{ m: 0, mt: 0.5, pl: 2, fontSize: 12 }}>
          {details.map((item) => (
            <li key={`${item.kind}:${item.mod}:${item.file ?? ''}`}>{item.detail}</li>
          ))}
        </Box>
      </Collapse>
    </Box>
  )
}

/** The open profile's durable change history, newest first, less the changes `hide` (still listed under New). */
export function EarlierChanges({ panel, hide }: { panel: Panel; hide: ReadonlySet<string> }) {
  const { t } = useLingui()
  const [target, setTarget] = useState<HistoryEvent | null>(null)
  const later = target ? laterEvents(panel.events, target.id) : []
  const undo = (ev: HistoryEvent) => {
    if (laterEvents(panel.events, ev.id).length === 0) {
      panel.revertTo(ev.id).catch(() => undefined)
    } else {
      setTarget(ev)
    }
  }
  if (panel.events.length === 0) {
    return (
      <Typography sx={{ p: 1.75, fontSize: 13, color: 'text.secondary' }}>
        {t`No changes yet`}
      </Typography>
    )
  }
  return (
    <>
      {panel.events
        .filter((ev) => !hide.has(ev.id))
        .map((ev) => (
          <EarlierRow key={ev.id} ev={ev} panel={panel} onUndo={undo} />
        ))}
      <ConfirmDialog
        open={target !== null}
        title={plural(later.length, {
          one: 'Undo this and # later change?',
          other: 'Undo this and # later changes?',
        })}
        confirmLabel={t`Undo`}
        color="warning"
        onCancel={() => setTarget(null)}
        onConfirm={() => {
          if (target) {
            panel.revertTo(target.id).catch(() => undefined)
          }
          setTarget(null)
        }}
      >
        <Typography
          sx={{ fontSize: 13, mb: 1 }}
        >{t`These changes will be reverted too:`}</Typography>
        {later.map((e) => (
          <Typography key={e.id} sx={{ fontSize: 13 }} color="text.secondary">
            {e.kind === 'good' ? t`Known good` : e.label}
          </Typography>
        ))}
      </ConfirmDialog>
      {panel.error === '' ? null : (
        <Typography sx={{ px: 1.5, py: 1, fontSize: 13, color: 'error.main' }}>
          {panel.error}
        </Typography>
      )}
    </>
  )
}
