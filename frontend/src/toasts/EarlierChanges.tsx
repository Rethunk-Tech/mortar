import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Collapse, Typography } from '@mui/material'
import { FileDiff, Undo2 } from 'lucide-react'
import { useState } from 'react'
import type { HistoryEvent } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { historyChangeSummary } from '../profiles/historyCounts.ts'
import { historyLabel } from '../profiles/historyLabel.ts'
import type { useHistoryPanel } from '../profiles/useHistoryPanel.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { laterEvents, undoTarget } from './history.ts'

type Panel = ReturnType<typeof useHistoryPanel>

function EarlierRow({
  ev,
  panel,
  onUndo,
}: {
  ev: HistoryEvent
  panel: Panel
  // Absent for the oldest event, which nothing older can undo.
  onUndo: ((ev: HistoryEvent) => void) | undefined
}) {
  const { t } = useLingui()
  const [diff, setDiff] = useState(false)
  const label = historyLabel(ev)
  const changes = historyChangeSummary(ev)
  const details = panel.items[ev.id] ?? []
  const trimmed = ev.kind === 'trimmed'
  return (
    <Box
      sx={{ px: space.pad, py: space.gap, borderBottom: '1px solid var(--mortar-hairline-faint)' }}
    >
      <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: space.gap }}>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography sx={{ fontSize: 'base.fontSize', fontWeight: 600 }}>{label}</Typography>
          <Typography sx={{ fontSize: 'tiny.fontSize', color: 'text.secondary' }}>
            {changes === '' ? null : `${changes} · `}
            <When value={ev.at} withTime={true} />
          </Typography>
        </Box>
        {details.length > 0 ? (
          <TipIconButton
            label={diff ? t`Hide changes` : t`Show changes`}
            aria-expanded={diff}
            onClick={() => setDiff(!diff)}
          >
            <FileDiff size={16} />
          </TipIconButton>
        ) : null}
        {trimmed || onUndo === undefined ? null : (
          <TipIconButton
            label={t`Undo this change: ${{ name: label }}`}
            disabled={panel.busy !== ''}
            onClick={() => onUndo(ev)}
          >
            <Undo2 size={16} />
          </TipIconButton>
        )}
      </Box>
      <Collapse in={diff} unmountOnExit={true}>
        <Box component="ul" sx={{ m: 0, mt: 0.5, pl: space.pad, fontSize: 'small.fontSize' }}>
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
      panel.revertTo(undoTarget(panel.events, ev.id)).catch(() => undefined)
    } else {
      setTarget(ev)
    }
  }
  if (panel.events.length === 0) {
    return (
      <Typography sx={{ p: space.pad, fontSize: 'base.fontSize', color: 'text.secondary' }}>
        {t`No changes yet`}
      </Typography>
    )
  }
  const listed = panel.events.filter((ev) => !hide.has(ev.id))
  return (
    <>
      {listed.slice(0, panel.shown).map((ev) => (
        <EarlierRow
          key={ev.id}
          ev={ev}
          panel={panel}
          onUndo={undoTarget(panel.events, ev.id) === '' ? undefined : undo}
        />
      ))}
      <ConfirmationDialog
        open={target !== null}
        title={t`Undo this and ${plural({ count: later.length }, { one: '# later change', other: '# later changes' })}?`}
        confirmLabel={t`Undo`}
        tone="warning"
        onCancel={() => setTarget(null)}
        onConfirm={() => {
          if (target) {
            panel.revertTo(undoTarget(panel.events, target.id)).catch(() => undefined)
          }
          setTarget(null)
        }}
      >
        <Typography
          sx={{ fontSize: 'base.fontSize', mb: 1 }}
        >{t`These changes will be reverted too:`}</Typography>
        {later.map((e) => (
          <Typography key={e.id} sx={{ fontSize: 'base.fontSize' }} color="text.secondary">
            {historyLabel(e)}
          </Typography>
        ))}
      </ConfirmationDialog>
      {panel.error === '' ? null : (
        <Typography
          sx={{ px: space.pad, py: space.gap, fontSize: 'base.fontSize', color: 'error.main' }}
        >
          {panel.error}
        </Typography>
      )}
    </>
  )
}
