import { TextField } from '@mui/material'
import { useState } from 'react'

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
  return (
    <TextField
      autoFocus={true}
      size={size}
      value={value}
      onChange={(e) => setValue(e.target.value)}
      onBlur={onCancel}
      onFocus={(e) => e.target.select()}
      onKeyDown={(e) => {
        if (e.key === 'Escape') {
          e.stopPropagation()
          onCancel()
        } else if (e.key === 'Enter' && value.trim()) {
          void onSubmit(value).then((ok) => ok && onCancel())
        }
      }}
      slotProps={{ htmlInput: { maxLength: 60, 'aria-label': label } }}
      sx={{ userSelect: 'text' }}
    />
  )
}
