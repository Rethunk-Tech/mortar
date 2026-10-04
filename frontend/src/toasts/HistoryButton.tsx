import { useLingui } from '@lingui/react/macro'
import { Badge, Box, Button, IconButton, Popover, Tooltip, Typography } from '@mui/material'
import { Bell } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { compact } from '../game/compact.ts'
import { When } from '../i18n/When.tsx'
import { useLaunch } from '../launch/store.ts'
import { isLocked } from '../mods/locked.ts'
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
  const status = useLaunch((s) => s.status)
  const startingProfile = useLaunch((s) => (s.starting ? s.startingProfile : ''))
  const { action } = item
  const locked =
    action?.profileId !== undefined && isLocked(status, action.profileId, startingProfile)
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
          <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{item.body}</Typography>
        ) : null}
        <Typography sx={{ fontSize: 11, color: 'text.secondary' }}>
          <When value={item.at} withTime={true} />
        </Typography>
      </Box>
      {action ? (
        <Tooltip title={state.disabled ? (state.reason ?? '') : ''}>
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

function HistoryPopover({
  open,
  anchorEl,
  history,
  onClose,
  onClear,
}: {
  open: boolean
  anchorEl: HTMLElement | null
  history: ToastHistoryItem[]
  onClose: () => void
  onClear: () => void
}) {
  const { t } = useLingui()
  return (
    <Popover
      open={open}
      anchorEl={anchorEl}
      onClose={onClose}
      anchorReference={anchorEl ? 'anchorEl' : 'anchorPosition'}
      anchorPosition={{ top: 80, left: 16 }}
      anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      transformOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      slotProps={{
        paper: {
          role: 'dialog',
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
        {history.length === 0 ? (
          <Typography sx={{ p: 1.75, fontSize: 13, color: 'text.secondary' }}>
            {t`No notifications yet`}
          </Typography>
        ) : (
          history.map((item) => <HistoryRow key={item.id} item={item} />)
        )}
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
  useEffect(() => {
    bellMounted = true
    return () => {
      bellMounted = false
    }
  }, [])
  const open = (el: HTMLElement) => {
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
        aria-label={t`Notifications`}
        aria-haspopup="dialog"
        aria-expanded={historyOpen}
        onClick={(e) => open(e.currentTarget)}
        sx={{ width: 40, height: 40, borderRadius: '6px' }}
      >
        <Badge badgeContent={unread} color="primary" max={99}>
          <Bell size={18} aria-hidden={true} />
        </Badge>
      </IconButton>
      <HistoryPopover
        open={historyOpen}
        anchorEl={anchor ?? bell.current}
        history={history}
        onClose={close}
        onClear={clearHistory}
      />
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
      onClose={() => setHistoryOpen(false)}
      onClear={clearHistory}
    />
  )
}
