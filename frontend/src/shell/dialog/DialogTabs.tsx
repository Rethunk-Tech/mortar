import { Box, Tab, Tabs } from '@mui/material'
import type { ReactNode } from 'react'

/** A form dialog's tab bar, for FormDialog's tabs slot; each tab controls the panel DialogPanel draws. */
export function DialogTabs<T extends string>({
  ids,
  tab,
  onTab,
  tabs,
  label,
}: {
  ids: string
  tab: T
  onTab: (tab: T) => void
  tabs: { id: T; label: string }[]
  label: string
}) {
  return (
    <Tabs
      value={tab}
      onChange={(_, next: T) => onTab(next)}
      aria-label={label}
      sx={{
        minHeight: 44,
        mx: -1.75,
        '& .MuiTabs-indicator': { height: 2 },
        '& .MuiTab-root': {
          minHeight: 44,
          minWidth: 0,
          px: '14px',
          fontSize: 'medium.fontSize',
          fontWeight: 400,
          color: 'text.secondary',
          '&.Mui-selected': { color: 'var(--mortar-ink)', fontWeight: 600 },
        },
      }}
    >
      {tabs.map((t) => (
        <Tab
          key={t.id}
          value={t.id}
          id={`${ids}-tab-${t.id}`}
          aria-controls={`${ids}-panel-${t.id}`}
          label={t.label}
        />
      ))}
    </Tabs>
  )
}

// Only the open panel is drawn; what a field holds lives in the dialog, so switching tabs loses nothing typed.
export function DialogPanel({
  ids,
  panel,
  open,
  children,
}: {
  ids: string
  panel: string
  open: boolean
  children: ReactNode
}) {
  return open ? (
    <Box
      role="tabpanel"
      id={`${ids}-panel-${panel}`}
      aria-labelledby={`${ids}-tab-${panel}`}
      // Room inside the scroll box for a slider's thumb at its far end.
      sx={{ overflowX: 'hidden', px: 1, mx: -1, display: 'flex', flexDirection: 'column', gap: 3 }}
    >
      {children}
    </Box>
  ) : null
}
