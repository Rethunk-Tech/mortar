import { Box, Chip, Tab, Tabs } from '@mui/material'
import type { ReactNode } from 'react'
import type { SectionTab } from '../mods/problemSection.ts'
import { space } from '../theme/density.ts'

// The Problems tab's header: a tab per section with its count, scrolling sideways when they do not fit, the chosen one
// filled in the primary colour. The chosen section's bulk action, when it has one, sits in a row of its own below,
// right-aligned; a section without one has no such row.
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
          minWidth: 0,
          '& .MuiTab-root': {
            flex: '0 0 auto',
            flexDirection: 'row',
            whiteSpace: 'nowrap',
            gap: space.gap,
            py: space.gap,
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
      {tabs.length > 0 && actions ? (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            height: 44,
            px: space.pad,
          }}
        >
          {actions}
        </Box>
      ) : null}
    </Box>
  )
}
