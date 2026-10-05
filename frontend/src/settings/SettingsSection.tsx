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
const ROW_GAP = 3
const BLOCK_GAP = 1.5

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
    <Box
      className="settings-section"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1,
        // A section whose rows all filtered out (each row renders nothing) hides with its heading.
        '&:has(> .settings-tiles:empty)': { display: 'none' },
        ...sx,
      }}
    >
      {title ? (
        <Box component="h3" sx={{ m: 0, mt: 1.5, mb: 0.5, fontSize: 18, fontWeight: 600 }}>
          {title}
        </Box>
      ) : null}
      {description ? <Box sx={{ fontSize: 14, color: 'text.secondary' }}>{description}</Box> : null}
      <Box
        className="settings-tiles"
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 1,
          containerType: 'inline-size',
          // Every setting is its own tile; a section is the heading above a stack of tiles.
          // Light mode uses opaque paper: a dark tint over the wallpaper reads as grey.
          '& > *': (theme) => ({
            bgcolor:
              theme.palette.mode === 'light'
                ? theme.palette.background.paper
                : 'var(--mortar-overlay-45)',
            borderRadius: '6px',
          }),
        }}
      >
        {children}
      </Box>
    </Box>
  )
}

// Content that is not a SettingRow takes part in search through its terms. Loose content outside a section
// marks itself so the search results know a page still has something to show.
export function Searchable({
  terms,
  loose = false,
  children,
}: {
  terms: string
  loose?: boolean
  children: ReactNode
}) {
  const query = useSettingsSearch()
  if (!prefMatches(query, terms)) {
    return null
  }
  return loose && query ? (
    <Box className="settings-match" sx={{ display: 'contents' }}>
      {children}
    </Box>
  ) : (
    children
  )
}

export function SettingRow({
  label,
  description,
  children,
  block = false,
}: {
  label: ReactNode
  description?: ReactNode
  children: ReactNode
  block?: boolean
}) {
  const query = useSettingsSearch()
  if (!prefMatches(query, nodeText(label), nodeText(description))) {
    return null
  }
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: block ? 'stretch' : 'center',
        flexDirection: block ? 'column' : 'row',
        gap: block ? BLOCK_GAP : ROW_GAP,
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
      {block ? (
        children
      ) : (
        <Box sx={{ flexShrink: 0, display: 'flex', alignItems: 'center' }}>{children}</Box>
      )}
    </Box>
  )
}
