import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, IconButton } from '@mui/material'
import { X } from 'lucide-react'
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
  const { action } = toast
  return (
    <Box
      role={toast.kind === 'error' || toast.kind === 'warning' ? 'alert' : 'status'}
      sx={{
        display: 'flex',
        alignItems: 'center',
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
      <Box sx={{ flexGrow: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        <Box component="span" sx={{ fontSize: 14, fontWeight: 600 }}>
          {toast.title}
        </Box>
        {toast.body && (
          <Box component="span" sx={{ fontSize: 13, color: 'rgba(235,235,240,0.95)' }}>
            {toast.body}
          </Box>
        )}
      </Box>
      {action && (
        <ButtonBase
          onClick={() => {
            action.run()
            dismiss(toast.id)
          }}
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
      )}
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
