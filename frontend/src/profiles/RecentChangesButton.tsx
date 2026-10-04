import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  IconButton,
  MenuItem,
  Popover,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
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
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { historyChangeSummary, historyEventKind } from './historyCounts.ts'
import { revertHistoryEvent } from './historyRevert.ts'
import { useRecentChanges } from './recentChanges.ts'

function RecentRow({
  ev,
  busyId,
  onUndo,
}: {
  ev: RecentEvent
  busyId: string
  onUndo: (profileId: string, eventId: string) => void
}) {
  const { t } = useLingui()
  const changes = historyChangeSummary(ev)
  const kind = historyEventKind(ev)
  let { label } = ev
  switch (kind) {
    case 'added':
      label = ev.label === 'added' ? t`Added` : ev.label
      break
    case 'removed':
      label = ev.label === 'removed' ? t`Removed` : ev.label
      break
    case 'updated':
      label = ev.label === 'updated' ? t`Updated` : ev.label
      break
    case 'enabled':
      label = ev.label === 'enabled' ? t`Switched on` : ev.label
      break
    case 'disabled':
      label = ev.label === 'disabled' ? t`Switched off` : ev.label
      break
    case 'pinned':
      label = ev.label === 'pinned' ? t`Pinned` : ev.label
      break
    case 'imported':
      label = ev.label === 'imported' ? t`Imported` : ev.label
      break
    case 'restored':
      label = ev.label === 'restored' ? t`Restored` : ev.label
      break
    case 'reverted':
      label = ev.label === 'reverted' ? t`Reverted` : ev.label
      break
    case 'bulk':
      label = ev.label === 'bulk' ? t`Bulk change` : ev.label
      break
    default:
      break
  }
  const thisBusy = busyId === ev.id
  const otherBusy = busyId !== '' && !thisBusy
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 1,
        px: 1.5,
        py: 1,
        borderBottom: '1px solid var(--mortar-hairline-faint)',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} title={ev.profileName} sx={{ fontSize: 13, fontWeight: 600 }}>
          {ev.profileName}
        </Typography>
        <Typography noWrap={true} title={label} sx={{ fontSize: 13 }}>
          {label}
        </Typography>
        <Typography sx={{ fontSize: 11, color: 'text.secondary' }}>
          {changes === '' ? null : `${changes} · `}
          <When value={ev.at} withTime={true} />
        </Typography>
      </Box>
      <DisabledReason title={t`Restoring…`} disabled={thisBusy || otherBusy}>
        <Button
          size="small"
          disabled={thisBusy || otherBusy}
          onClick={() => onUndo(ev.profileId, ev.id)}
          sx={{ whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Undo this change`}
        </Button>
      </DisabledReason>
    </Box>
  )
}

function RecentList({
  loaded,
  shown,
  busy,
  onUndo,
}: {
  loaded: boolean
  shown: RecentEvent[]
  busy: string
  onUndo: (profileId: string, eventId: string) => void
}) {
  const { t } = useLingui()
  if (!loaded) {
    return <LoadingRow>{t`Loading…`}</LoadingRow>
  }
  if (shown.length === 0) {
    return (
      <Typography sx={{ p: 1.75, fontSize: 13, color: 'text.secondary' }}>
        {t`No changes yet`}
      </Typography>
    )
  }
  return (
    <>
      {shown.map((ev) => (
        <RecentRow key={`${ev.profileId}:${ev.id}`} ev={ev} busyId={busy} onUndo={onUndo} />
      ))}
    </>
  )
}

function RecentPanel({
  open,
  anchor,
  events,
  loaded,
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
  loaded: boolean
  busy: string
  errorText: string
  missingEvent: string
  missingWants: Awaited<ReturnType<typeof revertHistoryEvent>>['missingWants']
  onClose: () => void
  onUndo: (profileId: string, eventId: string) => void
  onDownload: () => void
}) {
  const { t } = useLingui()
  const [profile, setProfile] = useState('')
  const names = [...new Map(events.map((ev) => [ev.profileId, ev.profileName])).entries()]
  const shown = profile === '' ? events : events.filter((ev) => ev.profileId === profile)
  return (
    <Popover
      open={open}
      anchorEl={anchor}
      onClose={onClose}
      anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      transformOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      slotProps={{
        paper: {
          role: 'dialog',
          sx: {
            width: 400,
            maxWidth: 'calc(100vw - 32px)',
            maxHeight: 440,
            bgcolor: 'var(--mortar-panel-solid)',
            border: '1px solid var(--mortar-hairline-12)',
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
          borderBottom: '1px solid var(--mortar-hairline-muted)',
        }}
      >
        <Typography sx={{ flex: 1, fontSize: 13, fontWeight: 700 }}>{t`Recent changes`}</Typography>
        {names.length > 1 ? (
          <TextField
            select={true}
            size="small"
            variant="standard"
            value={profile}
            onChange={(e) => setProfile(e.target.value)}
            slotProps={{ htmlInput: { 'aria-label': t`Show changes from` } }}
            sx={{ minWidth: 140, '& .MuiInputBase-root': { fontSize: 13 } }}
          >
            <MenuItem value="">{t`All profiles`}</MenuItem>
            {names.map(([id, name]) => (
              <MenuItem key={id} value={id}>
                {name}
              </MenuItem>
            ))}
          </TextField>
        ) : null}
      </Box>
      <Box sx={{ overflowY: 'auto', maxHeight: 380, [compact]: { maxHeight: 280 } }}>
        <RecentList loaded={loaded} shown={shown} busy={busy} onUndo={onUndo} />
      </Box>
      {errorText !== '' && (
        <Box sx={{ px: 1.5, py: 1, borderTop: '1px solid var(--mortar-hairline-muted)' }}>
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
  const open = useRecentChanges((s) => s.open)
  const setOpen = useRecentChanges((s) => s.setOpen)
  const [events, setEvents] = useState<RecentEvent[]>([])
  const [loaded, setLoaded] = useState(false)
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
      setLoaded(false)
      return
    }
    let live = true
    setLoaded(false)
    RecentHistory(game)
      .then((list) => {
        if (live) {
          setEvents(list ?? [])
          setLoaded(true)
        }
      })
      .catch((e: unknown) => {
        reportUnexpected(e)
        if (live) {
          setLoaded(true)
        }
      })
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
        loaded={loaded}
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
