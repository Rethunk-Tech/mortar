import { Box, Chip, Tab, Tabs } from '@mui/material'
import type { ReactNode } from 'react'
import type { SectionTab } from '../mods/problemSection.ts'

// The Problems tab's header row: a tab per section with its count, scrolling sideways when they do not fit beside
// the row's actions. The chosen tab is filled in the primary colour.
export function SectionStrip({
  tabs,
  current,
  onChoose,
  label,
  actions,
}: {
  tabs: readonly SectionTab[]
  current: string
  onChoose: (id: string) => void
  label: string
  actions: ReactNode
}) {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        bgcolor: 'var(--mortar-panel)',
        borderBottom: '1px solid var(--mortar-hairline)',
      }}
    >
      <Tabs
        value={tabs.some((tab) => tab.id === current) ? current : false}
        aria-label={label}
        variant="scrollable"
        allowScrollButtonsMobile={true}
        onChange={(_e, id: string) => onChoose(id)}
        slotProps={{ indicator: { sx: { display: 'none' } } }}
        sx={{
          flex: 1,
          minWidth: 0,
          '& .MuiTab-root': {
            flex: '1 0 auto',
            flexDirection: 'row',
            whiteSpace: 'nowrap',
            gap: 1,
            py: 1.25,
            maxWidth: 'none',
            textTransform: 'none',
            fontSize: 14,
            fontWeight: 400,
            color: 'text.primary',
            '&:hover': { bgcolor: 'action.hover' },
            '&.Mui-selected': {
              bgcolor: 'primary.main',
              color: 'primary.contrastText',
              '&:hover': { bgcolor: 'primary.dark' },
            },
          },
        }}
      >
        {tabs.map((tab) => (
          <Tab
            key={tab.id}
            value={tab.id}
            label={
              <>
                <Box component="span">{tab.label}</Box>
                <Chip
                  size="small"
                  component="span"
                  label={tab.count}
                  sx={{ bgcolor: 'var(--mortar-overlay-30)', color: 'inherit', fontSize: 12 }}
                />
              </>
            }
          />
        ))}
      </Tabs>
      <Box sx={{ display: 'flex', alignItems: 'center', flexShrink: 0, px: 1 }}>{actions}</Box>
    </Box>
  )
}
