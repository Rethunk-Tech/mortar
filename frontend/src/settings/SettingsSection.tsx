import { Box, type SxProps, type Theme } from '@mui/material'
import type { ReactNode } from 'react'
import { visuallyHidden } from '../shell/visuallyHidden.ts'
import { space } from '../theme/density.ts'
import { SEARCH_HIT } from './prefFilter.ts'
import { useSettingsMatch, useSettingsSearching } from './useSettingsSearch.ts'

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
  tiles = true,
}: {
  title?: ReactNode
  description?: ReactNode
  children: ReactNode
  sx?: SxProps<Theme>
  // Off for content that brings its own surfaces, such as a form or nested sections.
  tiles?: boolean
}) {
  const query = useSettingsSearching()
  return (
    <Box
      className="settings-section"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
        // A section whose rows all filtered out (each row renders nothing) hides with its heading. Rows sit
        // inside components (PrefKeys, sub-sections), so only the rendered result says whether any matched.
        '&:has(> .settings-tiles:empty)': { display: 'none' },
        // A section a Searchable wraps whole is a match itself.
        ...(query
          ? { [`&:not(:has(${SEARCH_HIT})):not(.settings-match > *)`]: { display: 'none' } }
          : {}),
        ...sx,
      }}
    >
      {title ? (
        <Box component="h3" sx={{ m: 0, mt: 1.5, mb: 0.5, fontSize: 18, fontWeight: 600 }}>
          {title}
        </Box>
      ) : null}
      {description ? (
        <Box sx={{ fontSize: 'medium.fontSize', color: 'text.secondary' }}>{description}</Box>
      ) : null}
      {tiles ? (
        <Box
          className="settings-tiles"
          sx={{
            display: 'flex',
            flexDirection: 'column',
            gap: space.gap,
            containerType: 'inline-size',
            // Controls inside a tile are filled blocks rather than outlined, so the tile reads as one surface.
            '& .MuiOutlinedInput-notchedOutline': { borderColor: 'transparent' },
            '& .MuiInputBase-root': { bgcolor: 'var(--mortar-raised)' },
            '& .MuiButton-outlined': {
              border: 0,
              bgcolor: 'var(--mortar-raised)',
              color: 'text.primary',
              '&:hover': { border: 0, bgcolor: 'var(--mortar-hairline-16)' },
            },
            // Every setting is its own tile; a section is the heading above a stack of tiles.
            // Light mode uses opaque paper: a dark tint over the wallpaper reads as grey.
            // Paper on a light wallpaper needs an edge to read as a tile. A settings-contents wrapper (SharedOnProfile)
            // is looked through, so what it wraps still tiles one by one.
            '& > :not(.settings-contents, .settings-match), & > .settings-match > *, & .settings-contents > :not(.settings-contents, .settings-caption)':
              (theme) =>
                theme.palette.mode === 'light'
                  ? {
                      bgcolor: theme.palette.background.paper,
                      border: '1px solid var(--mortar-hairline-muted)',
                      borderRadius: '6px',
                    }
                  : { bgcolor: 'var(--mortar-overlay-45)', borderRadius: '6px' },
          }}
        >
          {children}
        </Box>
      ) : (
        children
      )}
    </Box>
  )
}

// Content that is not a SettingRow takes part in search through its terms, and marks a match so its section and
// page stay in the results.
export function Searchable({ terms, children }: { terms: string; children: ReactNode }) {
  const matched = useSettingsMatch(terms)
  const searching = useSettingsSearching()
  if (!matched) {
    return null
  }
  return searching ? (
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
  hideTitle = false,
  mark,
}: {
  label: ReactNode
  description?: ReactNode
  children: ReactNode
  block?: boolean
  // For a row that is the whole section: the heading already says what it is, so the title stays only for screen
  // readers and search.
  hideTitle?: boolean
  // Rides after the label, outside the text search reads.
  mark?: ReactNode
}) {
  if (!useSettingsMatch(nodeText(label), nodeText(description))) {
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
        px: space.pad,
        py: space.gap,
        // A narrow card puts the control under its label instead of squeezing the label.
        [`@container (max-width: ${STACK_BELOW}px)`]: {
          flexDirection: 'column',
          alignItems: 'stretch',
          gap: space.gap,
        },
      }}
    >
      {/* With nothing to show, the label box leaves the layout entirely, so a stacked row gets no gap above it. */}
      <Box sx={hideTitle && !description ? visuallyHidden : { flex: 1, minWidth: 0 }}>
        <Box
          data-setting-label={true}
          sx={hideTitle ? visuallyHidden : { fontSize: 'large.fontSize' }}
        >
          {label}
          {mark}
        </Box>
        {description ? (
          <Box sx={{ fontSize: 'medium.fontSize', color: 'text.secondary', mt: 0.25 }}>
            {description}
          </Box>
        ) : null}
      </Box>
      {block ? (
        children
      ) : (
        <Box
          sx={{
            flexShrink: 0,
            display: 'flex',
            alignItems: 'center',
            // A Switch carries 12px of padding; pull it out so it ends where buttons and selects do.
            '& .MuiSwitch-root:last-child': { mr: -1.5 },
          }}
        >
          {children}
        </Box>
      )}
    </Box>
  )
}
