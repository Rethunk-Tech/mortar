import { Box, type SxProps, type Theme } from '@mui/material'
import { Children, isValidElement, type ReactNode } from 'react'
import { prefMatches, sectionVisible } from './prefFilter.ts'
import { useSettingsSearch } from './useSettingsSearch.ts'

function nodeText(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') {
    return String(node)
  }
  return ''
}

const STACK_BELOW = 560

export function SettingsSection({
  title,
  description,
  children,
  sx,
}: {
  title?: ReactNode
  description?: ReactNode
  children: ReactNode
  sx?: SxProps<Theme>
}) {
  const query = useSettingsSearch()
  const rows = Children.toArray(children).flatMap((child) => {
    if (!isValidElement(child)) {
      return []
    }
    const props = child.props as { label?: ReactNode; description?: ReactNode }
    if (props.label === undefined) {
      return []
    }
    return [{ label: nodeText(props.label), description: nodeText(props.description) }]
  })
  if (query && rows.length > 0 && !sectionVisible(query, rows)) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, ...sx }}>
      {title ? (
        <Box component="h3" sx={{ m: 0, mt: 1.5, mb: 0.5, fontSize: 18, fontWeight: 600 }}>
          {title}
        </Box>
      ) : null}
      {description ? <Box sx={{ fontSize: 14, color: 'text.secondary' }}>{description}</Box> : null}
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 1,
          containerType: 'inline-size',
          // Every setting is its own tile; a section is the heading above a stack of tiles.
          '& > *': { bgcolor: 'var(--mortar-overlay-45)', borderRadius: '6px' },
        }}
      >
        {children}
      </Box>
    </Box>
  )
}

export function SettingRow({
  label,
  description,
  children,
}: {
  label: ReactNode
  description?: ReactNode
  children: ReactNode
}) {
  const query = useSettingsSearch()
  if (!prefMatches(query, nodeText(label), nodeText(description))) {
    return null
  }
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 3,
        minHeight: 64,
        px: 2.5,
        py: 1.5,
        // Controls inside a row are filled blocks rather than outlined, so the tile reads as one surface.
        '& .MuiOutlinedInput-notchedOutline': { borderColor: 'transparent' },
        '& .MuiInputBase-root': { bgcolor: 'var(--mortar-raised)' },
        '& .MuiButton-outlined': {
          border: 0,
          bgcolor: 'var(--mortar-raised)',
          color: 'text.primary',
          '&:hover': { border: 0, bgcolor: 'var(--mortar-hairline-16)' },
        },
        // A narrow card puts the control under its label instead of squeezing the label.
        [`@container (max-width: ${STACK_BELOW}px)`]: {
          flexDirection: 'column',
          alignItems: 'stretch',
          gap: 1,
        },
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: 16 }}>{label}</Box>
        {description ? (
          <Box sx={{ fontSize: 14, color: 'text.secondary', mt: 0.25 }}>{description}</Box>
        ) : null}
      </Box>
      <Box sx={{ flexShrink: 0, display: 'flex', alignItems: 'center' }}>{children}</Box>
    </Box>
  )
}
