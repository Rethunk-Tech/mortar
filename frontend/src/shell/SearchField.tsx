import { InputAdornment, type SxProps, TextField, type Theme } from '@mui/material'
import { Search } from 'lucide-react'
import type { FocusEventHandler, KeyboardEventHandler, Ref } from 'react'

// The app's one search box: a magnifier, a tinted field and the label as both its accessible name and, unless a
// placeholder is given, its placeholder.
export function SearchField({
  label,
  placeholder,
  value,
  onChange,
  inputRef,
  autoFocus,
  fullWidth,
  onBlur,
  onKeyDown,
  sx,
}: {
  label: string
  placeholder?: string
  value: string
  onChange: (value: string) => void
  inputRef?: Ref<HTMLInputElement>
  autoFocus?: boolean
  fullWidth?: boolean
  onBlur?: FocusEventHandler<HTMLInputElement | HTMLTextAreaElement>
  onKeyDown?: KeyboardEventHandler<HTMLDivElement>
  sx?: SxProps<Theme>
}) {
  return (
    <TextField
      size="small"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder ?? label}
      inputRef={inputRef}
      autoFocus={autoFocus}
      fullWidth={fullWidth}
      onBlur={onBlur}
      onKeyDown={onKeyDown}
      slotProps={{
        htmlInput: { 'aria-label': label },
        input: {
          startAdornment: (
            <InputAdornment position="start">
              <Search size={14} aria-hidden={true} />
            </InputAdornment>
          ),
          sx: {
            height: 36,
            fontSize: 13,
            borderRadius: '6px',
            // Opaque paper in light mode: a dark tint over the wallpaper reads as grey.
            bgcolor: (theme) =>
              theme.palette.mode === 'light'
                ? theme.palette.background.paper
                : 'var(--mortar-overlay-30)',
            '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--mortar-hairline-15)' },
          },
        },
      }}
      sx={sx}
    />
  )
}
