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
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, ...sx }}>
      {title ? (
        <Box sx={{ fontSize: 13, fontWeight: 700, color: 'text.secondary' }}>{title}</Box>
      ) : null}
      {description ? <Box sx={{ fontSize: 13, color: 'text.secondary' }}>{description}</Box> : null}
      <Box
        sx={{
          bgcolor: 'var(--mortar-overlay-45)',
          borderRadius: 1,
          overflow: 'hidden',
          containerType: 'inline-size',
          '& > * + *': { borderTop: '1px solid var(--mortar-hairline)' },
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
        gap: 2,
        minHeight: 58,
        px: 2,
        py: 1,
        // A narrow card puts the control under its label instead of squeezing the label to an ellipsis.
        [`@container (max-width: ${STACK_BELOW}px)`]: {
          flexDirection: 'column',
          alignItems: 'stretch',
          gap: 1,
          '& .setting-description': { whiteSpace: 'normal' },
        },
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Box sx={{ fontSize: 14 }}>{label}</Box>
        {description ? (
          <Box
            className="setting-description"
            title={nodeText(description)}
            sx={{
              fontSize: 12,
              color: 'text.secondary',
              whiteSpace: 'nowrap',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
            }}
          >
            {description}
          </Box>
        ) : null}
      </Box>
      <Box sx={{ flexShrink: 0, display: 'flex', alignItems: 'center' }}>{children}</Box>
    </Box>
  )
}
