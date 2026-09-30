import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, IconButton, Tooltip } from '@mui/material'
import { ChevronDown, ChevronUp, X } from 'lucide-react'
import { useState } from 'react'
import { useLaunch } from '../launch/store.ts'
import { isLocked } from '../mods/locked.ts'
import { LetterTile } from '../mods/parts.tsx'
import { reportUnexpected } from './report.ts'
import { type Toast, type ToastKind, useToasts } from './store.ts'

const edge: Record<ToastKind, string> = {
  info: 'info.main',
  success: 'success.main',
  warning: 'warning.main',
  error: 'error.main',
}

function ToastCard({ toast }: { toast: Toast }) {
  const { t } = useLingui()
  const dismiss = useToasts((s) => s.dismiss)
  const hold = useToasts((s) => s.hold)
  const release = useToasts((s) => s.release)
  const status = useLaunch((s) => s.status)
  const startingProfile = useLaunch((s) => (s.starting ? s.startingProfile : ''))
  const [open, setOpen] = useState(false)
  const { action } = toast
  const locked =
    action?.profileId !== undefined && isLocked(status, action.profileId, startingProfile)
  const lockHint = t`Stop the game to change mods.`
  const run = () => {
    if (locked || !action) {
      return
    }
    Promise.resolve(action.run()).then((ok) => {
      if (ok !== false) {
        dismiss(toast.id)
      }
    }, reportUnexpected)
  }
  return (
    <Box
      onMouseEnter={() => hold(toast.id)}
      onMouseLeave={() => release(toast.id)}
      role={toast.kind === 'error' || toast.kind === 'warning' ? 'alert' : 'status'}
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: '12px',
        minHeight: 60,
        p: '8px 8px 8px 12px',
        bgcolor: 'rgba(30,30,36,0.98)',
        border: '1px solid rgba(255,255,255,0.12)',
        borderLeft: '4px solid',
        borderLeftColor: edge[toast.kind],
        borderRadius: '8px',
      }}
    >
      {toast.picture === undefined ? null : (
        <LetterTile mod={{ uniqueId: toast.title, name: toast.title, picture: toast.picture }} />
      )}
      <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        <Box component="span" sx={{ fontSize: 14, fontWeight: 600 }}>
          {toast.title}
        </Box>
        {toast.body ? (
          <Box component="span" sx={{ fontSize: 13, color: 'rgba(235,235,240,0.95)' }}>
            {toast.body}
          </Box>
        ) : null}
        {open && toast.detail ? (
          <Box
            component="pre"
            sx={{
              m: 0,
              mt: '6px',
              fontSize: 12,
              fontFamily: 'inherit',
              whiteSpace: 'pre-wrap',
              color: 'rgba(235,235,240,0.95)',
            }}
          >
            {toast.detail}
          </Box>
        ) : null}
      </Box>
      {toast.detail ? (
        <ButtonBase
          onClick={() => setOpen((v) => !v)}
          sx={{
            display: 'flex',
            alignItems: 'center',
            height: 34,
            px: '12px',
            mt: '4px',
            bgcolor: 'rgba(255,255,255,0.1)',
            borderRadius: '6px',
            color: '#ffffff',
            fontSize: 13,
            fontWeight: 600,
            fontFamily: 'inherit',
            whiteSpace: 'nowrap',
            gap: '4px',
          }}
        >
          {open ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
          {open ? t`Hide details` : t`Details`}
        </ButtonBase>
      ) : null}
      {action ? (
        <Tooltip title={locked ? lockHint : ''}>
          <span>
            <ButtonBase
              disabled={locked}
              onClick={run}
              sx={{
                height: 34,
                px: '12px',
                bgcolor: 'rgba(255,255,255,0.1)',
                borderRadius: '6px',
                color: '#ffffff',
                fontSize: 13,
                fontWeight: 600,
                fontFamily: 'inherit',
                whiteSpace: 'nowrap',
              }}
            >
              {action.label}
            </ButtonBase>
          </span>
        </Tooltip>
      ) : null}
      <IconButton
        aria-label={t`Dismiss`}
        onClick={() => dismiss(toast.id)}
        sx={{ width: 32, height: 32, color: 'rgba(235,235,240,0.95)' }}
      >
        <X size={14} />
      </IconButton>
    </Box>
  )
}

export function ToastHost() {
  const { t } = useLingui()
  const toasts = useToasts((s) => s.toasts)
  return (
    <Box
      role="region"
      aria-label={t`Notifications`}
      sx={{
        position: 'fixed',
        right: 20,
        bottom: 20,
        width: 400,
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        zIndex: 'tooltip',
      }}
    >
      {toasts.map((toast) => (
        <ToastCard key={toast.id} toast={toast} />
      ))}
    </Box>
  )
}
