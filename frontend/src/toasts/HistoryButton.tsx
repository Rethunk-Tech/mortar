import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Badge, Box, Button, IconButton, Popover, Tooltip, Typography } from '@mui/material'
import { Bell } from 'lucide-react'
import { type ReactNode, useEffect, useRef, useState } from 'react'
import { compact } from '../game/compact.ts'
import { When } from '../i18n/When.tsx'
import { useProfileLocked } from '../mods/useLocked.ts'
import { HistoryDialog } from '../profiles/HistoryDialog.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useHistoryPanel } from '../profiles/useHistoryPanel.ts'
import { useQueue } from '../queue/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { closeTitleMenu } from '../shell/titleMenus.ts'
import { EarlierChanges } from './EarlierChanges.tsx'
import { historyActionState } from './history.ts'
import { reportUnexpected } from './report.ts'
import { type ToastHistoryItem, useToasts } from './store.ts'

const edge: Record<ToastHistoryItem['kind'], string> = {
  info: 'info.main',
  success: 'success.main',
  warning: 'warning.main',
  error: 'error.main',
}

function HistoryRow({ item }: { item: ToastHistoryItem }) {
  const { t } = useLingui()
  const { action } = item
  const profileHeld = useProfileLocked(action?.profileId ?? '')
  const locked = action?.profileId !== undefined && profileHeld
  const lockHint = t`Stop the game to change mods.`
  const state = historyActionState(action?.live, locked, lockHint)
  const run = () => {
    if (state.disabled || !action) {
      return
    }
    Promise.resolve(action.run()).catch(reportUnexpected)
  }
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 1,
        px: 1.5,
        py: 1,
        borderLeft: '3px solid',
        borderLeftColor: edge[item.kind],
      }}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, flex: 1, minWidth: 0 }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600 }}>
          {item.title}
          {item.count && item.count > 1 ? ` (×${item.count})` : ''}
        </Typography>
        {item.body ? (
          <Typography sx={{ fontSize: 12, color: 'text.secondary', overflowWrap: 'anywhere' }}>
            {item.body}
          </Typography>
        ) : null}
        <Typography sx={{ fontSize: 11, color: 'text.secondary' }}>
          <When value={item.at} withTime={true} />
        </Typography>
      </Box>
      {action ? (
        <Tooltip title={state.disabled ? (state.reason ?? '') : ''} describeChild={true}>
          <span>
            <Button size="small" disabled={state.disabled} onClick={run} sx={{ flexShrink: 0 }}>
              {action.label}
            </Button>
          </span>
        </Tooltip>
      ) : null}
    </Box>
  )
}

let bellMounted = false

function SectionLabel({ children }: { children: ReactNode }) {
  return (
    <Typography sx={{ px: 1.5, pt: 1, fontSize: 11, fontWeight: 700, color: 'text.secondary' }}>
      {children}
    </Typography>
  )
}

function HistoryPopover({
  open,
  anchorEl,
  history,
  freshCount,
  onClose,
  onClear,
  onAll,
}: {
  open: boolean
  anchorEl: HTMLElement | null
  history: ToastHistoryItem[]
  freshCount: number
  onClose: () => void
  onClear: () => void
  onAll: () => void
}) {
  const { t } = useLingui()
  const profileId = useProfiles((st) => st.openId)
  const panel = useHistoryPanel(profileId, open, true)
  // A change to the profile is listed once: while unread under New, then in the history under Earlier.
  const fresh = history.slice(0, freshCount)
  const reported = new Set(fresh.flatMap((item) => item.changes ?? []))
  const readNotes = history.slice(freshCount).filter((item) => item.action?.profileId === undefined)
  return (
    <Popover
      open={open}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorReference={anchorEl ? 'anchorEl' : 'anchorPosition'}
      anchorPosition={{ top: 80, left: 16 }}
      anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      transformOrigin={{ vertical: anchorEl ? 'top' : 'bottom', horizontal: 'right' }}
      slotProps={{
        paper: {
          role: 'dialog',
          'aria-label': t`Notification history`,
          sx: {
            width: 360,
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
        <Typography sx={{ flex: 1, fontSize: 13, fontWeight: 700 }}>{t`Notifications`}</Typography>
        <Button size="small" onClick={onClear} disabled={history.length === 0}>
          {t`Clear`}
        </Button>
      </Box>
      <Box sx={{ overflowY: 'auto', maxHeight: 380, [compact]: { maxHeight: 280 } }}>
        {history.length === 0 && panel.events.length === 0 ? (
          <EmptyState compact={true} icon={<Bell size={28} />} title={t`No notifications yet`}>
            {t`Results of installs, updates and launches appear here.`}
          </EmptyState>
        ) : (
          <>
            <SectionLabel>{t`New`}</SectionLabel>
            {fresh.length === 0 ? (
              <Typography sx={{ px: 1.5, pb: 1, fontSize: 13, color: 'text.secondary' }}>
                {t`Nothing new`}
              </Typography>
            ) : (
              fresh.map((item) => <HistoryRow key={item.id} item={item} />)
            )}
            <SectionLabel>{t`Earlier`}</SectionLabel>
            {readNotes.map((item) => (
              <HistoryRow key={item.id} item={item} />
            ))}
            <EarlierChanges panel={panel} hide={reported} />
          </>
        )}
      </Box>
      <Box
        sx={{
          display: 'flex',
          borderTop: '1px solid var(--mortar-hairline-muted)',
          px: 1.5,
          py: 0.5,
        }}
      >
        <Button size="small" disabled={!panel.game || profileId === ''} onClick={onAll}>
          {t`All changes…`}
        </Button>
        <Button
          size="small"
          sx={{ ml: 'auto' }}
          onClick={() => {
            onClose()
            useQueue.getState().setOpen(true)
          }}
        >
          {t`Downloads…`}
        </Button>
      </Box>
    </Popover>
  )
}

export function HistoryButton() {
  const { t } = useLingui()
  const historyOpen = useToasts((s) => s.historyOpen)
  const setHistoryOpen = useToasts((s) => s.setHistoryOpen)
  const history = useToasts((s) => s.history)
  const unread = useToasts((s) => s.unread)
  const clearHistory = useToasts((s) => s.clearHistory)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const bell = useRef<HTMLButtonElement>(null)
  const openId = useProfiles((st) => st.openId)
  const [allOpen, setAllOpen] = useState(false)
  // Opening marks everything read, so the unread count is kept for the length of the visit.
  const unreadBefore = useRef(0)
  if (!historyOpen) {
    unreadBefore.current = unread
  }
  useEffect(() => {
    bellMounted = true
    return () => {
      bellMounted = false
    }
  }, [])
  const open = (el: HTMLElement) => {
    closeTitleMenu()
    setAnchor(el)
    setHistoryOpen(true)
  }
  const close = () => {
    setAnchor(null)
    setHistoryOpen(false)
  }
  return (
    <>
      <IconButton
        ref={bell}
        aria-label={
          unread > 0
            ? plural(unread, { one: 'Notifications, # unread', other: 'Notifications, # unread' })
            : t`Notifications`
        }
        aria-haspopup="dialog"
        aria-expanded={historyOpen}
        onClick={(e) => open(e.currentTarget)}
        sx={{ '--wails-draggable': 'no-drag', width: 36, height: 34, borderRadius: '8px' }}
      >
        <Badge
          // None rather than 0: a hidden badge still holds its 0, text the button's name would then lack.
          badgeContent={unread || null}
          color="primary"
          max={99}
          slotProps={{ badge: { 'aria-hidden': true } }}
        >
          <Bell size={17} aria-hidden={true} />
        </Badge>
      </IconButton>
      <HistoryPopover
        open={historyOpen}
        anchorEl={anchor ?? bell.current}
        history={history}
        freshCount={historyOpen ? unreadBefore.current : 0}
        onClose={close}
        onClear={clearHistory}
        onAll={() => {
          close()
          setAllOpen(true)
        }}
      />
      <HistoryDialog profileId={openId} open={allOpen} onClose={() => setAllOpen(false)} />
    </>
  )
}

export function HistoryFallback() {
  const historyOpen = useToasts((s) => s.historyOpen)
  const setHistoryOpen = useToasts((s) => s.setHistoryOpen)
  const history = useToasts((s) => s.history)
  const clearHistory = useToasts((s) => s.clearHistory)
  return (
    <HistoryPopover
      open={historyOpen && !bellMounted}
      anchorEl={null}
      history={history}
      freshCount={0}
      onClose={() => setHistoryOpen(false)}
      onClear={clearHistory}
      onAll={() => setHistoryOpen(false)}
    />
  )
}
