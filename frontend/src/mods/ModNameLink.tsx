import { Link, type SxProps, type Theme } from '@mui/material'
import { openModHandlers } from './openMod.ts'

// ModNameLink is a mod's name that opens its details on click or right-click, like a row on the Mods tab.
export function ModNameLink({
  id,
  modKey,
  name,
  wrap = false,
  sx,
}: {
  id: string
  modKey?: string
  name: string
  // wrap lets a long name break across lines instead of ending in an ellipsis.
  wrap?: boolean
  sx?: SxProps<Theme>
}) {
  return (
    <Link
      component="button"
      underline="hover"
      color="inherit"
      title={name}
      {...openModHandlers(modKey ? { id, key: modKey } : { id })}
      sx={[
        { font: 'inherit', textAlign: 'left', minWidth: 0 },
        wrap
          ? { overflowWrap: 'anywhere' }
          : { overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
        ...(Array.isArray(sx) ? sx : [sx]),
      ]}
    >
      {name}
    </Link>
  )
}
