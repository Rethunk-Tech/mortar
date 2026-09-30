import { useLingui } from '@lingui/react/macro'
import { Badge, Box, Button, IconButton, Popover, Tooltip, Typography } from '@mui/material'
import { Bell } from 'lucide-react'
import { useState } from 'react'
import { compact } from '../game/compact.ts'
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
  const { t, i18n } = useLingui()
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
        flexDirection: 'column',
        gap: 0.5,
        px: 1.5,
        py: 1,
        borderLeft: '3px solid',
        borderLeftColor: edge[item.kind],
      }}
    >
      <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{item.title}</Typography>
      {item.body ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{item.body}</Typography>
      ) : null}
      <Typography sx={{ fontSize: 11, color: 'text.secondary' }}>
        {new Intl.DateTimeFormat(i18n.locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
          new Date(item.at),
        )}
      </Typography>
      {action ? (
        <Tooltip title={state.disabled ? (state.reason ?? '') : ''}>
          <span>
            <Button
              size="small"
              disabled={state.disabled}
              onClick={run}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {action.label}
            </Button>
          </span>
        </Tooltip>
      ) : null}
    </Box>
  )
}

export function HistoryButton() {
  const { t } = useLingui()
  const history = useToasts((s) => s.history)
  const unread = useToasts((s) => s.unread)
  const markRead = useToasts((s) => s.markRead)
  const clearHistory = useToasts((s) => s.clearHistory)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const open = (el: HTMLElement) => {
    setAnchor(el)
    markRead()
  }
  return (
    <>
      <IconButton
        aria-label={t`Notifications`}
        aria-haspopup="dialog"
        aria-expanded={anchor !== null}
        onClick={(e) => open(e.currentTarget)}
        sx={{ width: 40, height: 40, borderRadius: '6px' }}
      >
        <Badge badgeContent={unread} color="primary" max={99}>
          <Bell size={18} aria-hidden={true} />
        </Badge>
      </IconButton>
      <Popover
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        transitionDuration={0}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
        transformOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        slotProps={{
          paper: {
            role: 'dialog',
            sx: {
              width: 360,
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
          <Typography
            sx={{ flex: 1, fontSize: 13, fontWeight: 700 }}
          >{t`Notifications`}</Typography>
          <Button
            size="small"
            onClick={clearHistory}
            disabled={history.length === 0}
            sx={{ whiteSpace: 'nowrap' }}
          >
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
    </>
  )
}
