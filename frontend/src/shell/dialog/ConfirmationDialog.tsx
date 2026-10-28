import { useLingui } from '@lingui/react/macro'
import { Button, Checkbox, CircularProgress, FormControlLabel } from '@mui/material'
import { Info, OctagonAlert, TriangleAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import { AppDialog, DialogFooter, type DialogTone, ToneIcon } from './AppDialog.tsx'

const ICONS: Record<DialogTone, ReactNode> = {
  neutral: <Info size={22} />,
  warning: <TriangleAlert size={22} />,
  danger: <OctagonAlert size={22} />,
}

/**
 * Asks before an action: the title is the question, the body says what happens.
 * The action's button names the action ("Delete profile"), never "OK"; a dangerous one is red and Cancel takes focus.
 */
export function ConfirmationDialog({
  open,
  tone = 'neutral',
  title,
  body,
  confirmLabel,
  cancelLabel,
  busy = false,
  confirmDisabled = false,
  fieldFocus = false,
  checkbox,
  onConfirm,
  onCancel,
  children,
}: {
  open: boolean
  tone?: DialogTone
  title: ReactNode
  body?: ReactNode
  confirmLabel: ReactNode
  cancelLabel?: ReactNode
  busy?: boolean
  confirmDisabled?: boolean | undefined
  // The body holds a field that takes initial focus instead of a button.
  fieldFocus?: boolean
  checkbox?: { label: string; checked: boolean; onChange: (on: boolean) => void }
  onConfirm: () => void
  onCancel: () => void
  // More than the body can say: a choice, a field, a list to review.
  children?: ReactNode
}) {
  const { t } = useLingui()
  // A closed confirmation draws nothing, so the screen that holds it does not repaint a dialog tree on every render.
  if (!open) {
    return null
  }
  return (
    <AppDialog
      open={open}
      onClose={busy ? () => undefined : onCancel}
      closable={false}
      size="md"
      icon={<ToneIcon tone={tone}>{ICONS[tone]}</ToneIcon>}
      title={title}
      subtitle={body}
      footer={
        <DialogFooter
          aside={
            checkbox ? (
              <FormControlLabel
                control={
                  <Checkbox
                    size="small"
                    checked={checkbox.checked}
                    onChange={(_, on) => checkbox.onChange(on)}
                  />
                }
                label={checkbox.label}
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
      {children}
    </AppDialog>
  )
}
