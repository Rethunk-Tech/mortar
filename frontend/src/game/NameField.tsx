import { TextField } from '@mui/material'
import { useState } from 'react'
import { errorMessage } from '../toasts/report.ts'

// Enter submits, Esc or blur cancels; `onSubmit` reports whether the name was accepted.
export function NameField({
  initial,
  label,
  onSubmit,
  onCancel,
  size,
}: {
  initial: string
  label: string
  onSubmit: (name: string) => Promise<boolean>
  onCancel: () => void
  size?: 'small'
}) {
  const [value, setValue] = useState(initial)
  const [error, setError] = useState('')
  return (
    <TextField
      autoFocus={true}
      size={size}
      value={value}
      onChange={(e) => {
        setValue(e.target.value)
        setError('')
      }}
      error={error !== ''}
      helperText={error || undefined}
      onBlur={onCancel}
      onFocus={(e) => e.target.select()}
      onKeyDown={(e) => {
        if (e.key === 'Escape') {
          e.stopPropagation()
          onCancel()
        } else if (e.key === 'Enter' && value.trim()) {
          onSubmit(value)
            .then((ok) => ok && onCancel())
            .catch((err) => setError(errorMessage(err)))
        }
      }}
      slotProps={{ htmlInput: { maxLength: 60, 'aria-label': label } }}
      sx={{ userSelect: 'text' }}
    />
  )
}
