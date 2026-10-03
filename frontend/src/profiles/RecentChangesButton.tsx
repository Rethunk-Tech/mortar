import { useLingui } from '@lingui/react/macro'
import { Box, Button, IconButton, Popover, Tooltip, Typography } from '@mui/material'
import { History as HistoryIcon } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { RecentEvent } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  History,
  RecentHistory,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { compact } from '../game/compact.ts'
import { When } from '../i18n/When.tsx'
import { download } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { historyChangeSummary } from './historyCounts.ts'
import { revertHistoryEvent } from './historyRevert.ts'

function RecentRow({
  ev,
  busy,
  onUndo,
}: {
  ev: RecentEvent
  busy: boolean
  onUndo: (profileId: string, eventId: string) => void
}) {
  const { t } = useLingui()
  const changes = historyChangeSummary(ev)
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 1,
        px: 1.5,
        py: 1,
        borderBottom: '1px solid rgba(255,255,255,0.06)',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{ev.profileName}</Typography>
        <Typography sx={{ fontSize: 13 }}>{ev.label}</Typography>
        <Typography sx={{ fontSize: 11, color: 'text.secondary' }}>
          {changes === '' ? null : `${changes} · `}
          <When value={ev.at} withTime={true} />
        </Typography>
      </Box>
      <Button
        size="small"
        disabled={busy}
        onClick={() => onUndo(ev.profileId, ev.id)}
        sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Undo`}
      </Button>
    </Box>
  )
}

function RecentPanel({
  open,
  anchor,
  events,
  busy,
  errorText,
  missingEvent,
  missingWants,
  onClose,
  onUndo,
  onDownload,
}: {
  open: boolean
  anchor: HTMLElement | null
  events: RecentEvent[]
  busy: string
  errorText: string
  missingEvent: string
  missingWants: Awaited<ReturnType<typeof revertHistoryEvent>>['missingWants']
  onClose: () => void
  onUndo: (profileId: string, eventId: string) => void
  onDownload: () => void
}) {
  const { t } = useLingui()
  return (
    <Popover
      open={open}
      anchorEl={anchor}
      onClose={onClose}
      transitionDuration={0}
      anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      transformOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      slotProps={{
        paper: {
          role: 'dialog',
          sx: {
            width: 400,
            maxWidth: 'calc(100vw - 32px)',
            maxHeight: 440,
            bgcolor: 'rgb(40,40,48)',
            backgroundImage: 'none',
            border: '1px solid rgba(255,255,255,0.12)',
            borderRadius: '8px',
          },
        },
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          px: 1.5,
          py: 1,
          borderBottom: '1px solid rgba(255,255,255,0.08)',
        }}
      >
        <Typography sx={{ flex: 1, fontSize: 13, fontWeight: 700 }}>{t`Recent changes`}</Typography>
      </Box>
      <Box sx={{ overflowY: 'auto', maxHeight: 380, [compact]: { maxHeight: 280 } }}>
        {events.length === 0 ? (
          <Typography sx={{ p: 1.75, fontSize: 13, color: 'text.secondary' }}>
            {t`No changes yet`}
          </Typography>
        ) : (
          events.map((ev) => (
            <RecentRow
              key={`${ev.profileId}:${ev.id}`}
              ev={ev}
              busy={busy !== ''}
              onUndo={onUndo}
            />
          ))
        )}
      </Box>
      {errorText !== '' && (
        <Box sx={{ px: 1.5, py: 1, borderTop: '1px solid rgba(255,255,255,0.08)' }}>
          <Typography sx={{ fontSize: 13, color: 'error.main' }}>{errorText}</Typography>
          {missingEvent !== '' && missingWants.length > 0 && (
            <Button
              size="small"
              sx={{ mt: 1, whiteSpace: 'nowrap' }}
              disabled={busy !== ''}
              onClick={onDownload}
            >
              {t`Download missing mods`}
            </Button>
          )}
        </Box>
      )}
    </Popover>
  )
}

export function RecentChangesButton({ game }: { game: string }) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const [events, setEvents] = useState<RecentEvent[]>([])
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [missingEvent, setMissingEvent] = useState('')
  const [missingNames, setMissingNames] = useState<string[]>([])
  const [missingWants, setMissingWants] = useState<
    Awaited<ReturnType<typeof revertHistoryEvent>>['missingWants']
  >([])
  const btn = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) {
      return
    }
    let live = true
    RecentHistory(game)
      .then((list) => {
        if (live) {
          setEvents(list ?? [])
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [open, game])

  const revertTo = async (profileId: string, eventId: string) => {
    setBusy(eventId)
    const profileEvents = (await History(game, profileId)) ?? []
    const result = await revertHistoryEvent(game, profileId, eventId, profileEvents)
    setError(result.error)
    setMissingEvent(result.missingEvent)
    setMissingNames(result.missingNames)
    setMissingWants(result.missingWants)
    setEvents((await RecentHistory(game)) ?? [])
    setBusy('')
  }
  const errorText =
    missingNames.length > 0 ? t`Could not restore ${missingNames.join(', ')}` : error

  return (
    <>
      <Tooltip title={t`Recent changes`}>
        <IconButton
          ref={btn}
          aria-label={t`Recent changes`}
          aria-haspopup="dialog"
          aria-expanded={open}
          onClick={() => setOpen(true)}
          sx={{ width: 40, height: 40, borderRadius: '6px' }}
        >
          <HistoryIcon size={18} aria-hidden={true} />
        </IconButton>
      </Tooltip>
      <RecentPanel
        open={open}
        anchor={btn.current}
        events={events}
        busy={busy}
        errorText={errorText}
        missingEvent={missingEvent}
        missingWants={missingWants}
        onClose={() => setOpen(false)}
        onUndo={(profileId, eventId) => revertTo(profileId, eventId).catch(reportUnexpected)}
        onDownload={() => download(missingWants).catch(reportUnexpected)}
      />
    </>
  )
}
