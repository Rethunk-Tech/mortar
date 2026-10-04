import { ButtonBase, type ButtonBaseProps } from '@mui/material'

// A row's whole-row button, laid under its content so the row can also hold its own controls without nesting
// them inside a button. The row is position: relative; its text ignores pointer events and its controls sit
// above with position: relative.
export function CoverButton({ sx, ...props }: Omit<ButtonBaseProps, 'children'>) {
  return (
    <ButtonBase
      {...props}
      sx={[
        { position: 'absolute', inset: 0, borderRadius: 'inherit' },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    />
  )
}
