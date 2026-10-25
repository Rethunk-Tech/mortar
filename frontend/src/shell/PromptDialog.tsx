import { useLingui } from '@lingui/react/macro'
import { useEffect, useState } from 'react'
import { FormDialog, TextInput } from './dialog/FormDialog.tsx'

/** One text field: Cancel and a contained confirm; Enter submits. */
export function PromptDialog({
  open,
  title,
  label,
  initial = '',
  maxLength,
  confirmLabel,
  allowEmpty = false,
  busy = false,
  onCancel,
  onSubmit,
}: {
  open: boolean
  title: string
  label: string
  initial?: string
  maxLength?: number
  confirmLabel: string
  allowEmpty?: boolean
  busy?: boolean
  onCancel: () => void
  onSubmit: (value: string) => void
}) {
  const { t } = useLingui()
  const [value, setValue] = useState(initial)
  useEffect(() => {
    if (open) {
      setValue(initial)
    }
  }, [open, initial])
  const trimmed = value.trim()
  return (
    <FormDialog
      open={open}
      onClose={onCancel}
      title={title}
      size="sm"
      submitLabel={confirmLabel}
      issue={allowEmpty || trimmed !== '' ? '' : t`${label} is empty.`}
      // One line is quicker to retype than a question about discarding it.
      dirty={false}
      busy={busy}
      onSubmit={() => onSubmit(trimmed)}
    >
      <TextInput
        label={label}
        value={value}
        onChange={setValue}
        autoFocus={true}
        {...(maxLength === undefined ? {} : { maxLength })}
      />
    </FormDialog>
  )
}
