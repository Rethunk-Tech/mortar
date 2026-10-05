import { useLingui } from '@lingui/react/macro'
import { Box, Button, Collapse, Typography } from '@mui/material'
import { useState } from 'react'
import type { HistoryEvent } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { When } from '../i18n/When.tsx'
import { historyChangeSummary } from '../profiles/historyCounts.ts'
import type { useHistoryPanel } from '../profiles/useHistoryPanel.ts'

type Panel = ReturnType<typeof useHistoryPanel>

function EarlierRow({ ev, panel }: { ev: HistoryEvent; panel: Panel }) {
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
          <Button
            size="small"
            disabled={panel.busy !== ''}
            onClick={() => panel.revertTo(ev.id).catch(() => undefined)}
          >
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

/** The open profile's durable change history, newest first. */
export function EarlierChanges({ panel }: { panel: Panel }) {
  const { t } = useLingui()
  if (panel.events.length === 0) {
    return (
      <Typography sx={{ p: 1.75, fontSize: 13, color: 'text.secondary' }}>
        {t`No changes yet`}
      </Typography>
    )
  }
  return (
    <>
      {panel.events.map((ev) => (
        <EarlierRow key={ev.id} ev={ev} panel={panel} />
      ))}
      {panel.error === '' ? null : (
        <Typography sx={{ px: 1.5, py: 1, fontSize: 13, color: 'error.main' }}>
          {panel.error}
        </Typography>
      )}
    </>
  )
}
