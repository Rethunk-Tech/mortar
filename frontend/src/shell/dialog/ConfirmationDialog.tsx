import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Checkbox,
  CircularProgress,
  FormControlLabel,
  Typography,
} from '@mui/material'
import { Info, OctagonAlert, TriangleAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import { AppDialog, DialogFooter, type DialogTone, ToneIcon } from './AppDialog.tsx'

const ICONS: Record<DialogTone, ReactNode> = {
  neutral: <Info size={22} />,
  warning: <TriangleAlert size={22} />,
  danger: <OctagonAlert size={22} />,
}

/**
 * Asks before an action: the title is the question, the body says what happens, and consequences list what changes.
 * The action's button names the action ("Delete profile"), never "OK"; a dangerous one is red and Cancel takes focus.
 */
export function ConfirmationDialog({
  open,
  tone = 'neutral',
  title,
  body,
  consequences = [],
  confirmLabel,
  cancelLabel,
  busy = false,
  confirmDisabled = false,
  fieldFocus = false,
  dontAskAgain,
  onConfirm,
  onCancel,
  children,
}: {
  open: boolean
  tone?: DialogTone
  title: ReactNode
  body?: ReactNode
  consequences?: string[]
  confirmLabel: ReactNode
  cancelLabel?: ReactNode
  busy?: boolean
  confirmDisabled?: boolean | undefined
  // The body holds a field that takes initial focus instead of a button.
  fieldFocus?: boolean
  dontAskAgain?: { label: string; checked: boolean; onChange: (on: boolean) => void }
  onConfirm: () => void
  onCancel: () => void
  // More than the consequences can say: a choice, a field, a list to review.
  children?: ReactNode
}) {
  const { t } = useLingui()
  // A closed confirmation draws nothing, so the screen that holds it does not repaint a dialog tree on every render.
  if (!open) {
    return null
  }
  const consequenceList =
    consequences.length === 0 ? null : (
      <Box
        component="ul"
        sx={{
          m: 0,
          px: 2,
          py: 1.25,
          listStyle: 'none',
          borderRadius: '8px',
          bgcolor: 'var(--mortar-raised)',
          display: 'flex',
          flexDirection: 'column',
          gap: 0.5,
        }}
      >
        {consequences.map((c) => (
          <Typography key={c} component="li" sx={{ fontSize: 'medium.fontSize' }}>
            {c}
          </Typography>
        ))}
      </Box>
    )
  return (
    <AppDialog
      open={open}
      onClose={onCancel}
      closable={false}
      size="md"
      icon={<ToneIcon tone={tone}>{ICONS[tone]}</ToneIcon>}
      title={title}
      subtitle={body}
      footer={
        <DialogFooter
          aside={
            dontAskAgain ? (
              <FormControlLabel
                control={
                  <Checkbox
                    size="small"
                    checked={dontAskAgain.checked}
                    onChange={(_, on) => dontAskAgain.onChange(on)}
                  />
                }
                label={dontAskAgain.label}
                sx={{ m: 0, '& .MuiFormControlLabel-label': { fontSize: 'small.fontSize' } }}
              />
            ) : null
          }
        >
          <Button onClick={onCancel} disabled={busy} autoFocus={!fieldFocus && tone === 'danger'}>
            {cancelLabel ?? t`Cancel`}
          </Button>
          <Button
            variant="contained"
            color={tone === 'danger' ? 'error' : 'primary'}
            onClick={onConfirm}
            disabled={busy || confirmDisabled}
            autoFocus={!fieldFocus && tone !== 'danger'}
            startIcon={busy ? <CircularProgress size={14} color="inherit" /> : null}
          >
            {confirmLabel}
          </Button>
        </DialogFooter>
      }
    >
      {consequenceList || children ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {consequenceList}
          {children}
        </Box>
      ) : null}
    </AppDialog>
  )
}
