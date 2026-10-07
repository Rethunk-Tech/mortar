import { Box, ToggleButton, ToggleButtonGroup } from '@mui/material'
import type { KeyboardEvent, ReactNode } from 'react'
import { type SectionTab, stepSection } from '../mods/problemSection.ts'

// The Problems tab's header row: an exclusive segment per section with its count, scrolling sideways when they do not
// fit beside the row's actions. Left and Right move between segments.
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
  const onKeyDown = (e: KeyboardEvent<HTMLElement>) => {
    const delta = { ArrowLeft: -1, ArrowRight: 1 }[e.key]
    if (delta === undefined) {
      return
    }
    e.preventDefault()
    const next = stepSection(tabs, current, delta)
    onChoose(next)
    e.currentTarget.querySelector<HTMLElement>(`[data-section="${next}"]`)?.focus()
  }
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        bgcolor: 'var(--mortar-panel)',
        borderBottom: '1px solid var(--mortar-hairline)',
      }}
    >
      <ToggleButtonGroup
        exclusive={true}
        value={current}
        aria-label={label}
        onKeyDown={onKeyDown}
        onChange={(_e, id: string | null) => {
          if (id !== null) {
            onChoose(id)
          }
        }}
        sx={{
          flex: 1,
          minWidth: 0,
          overflowX: 'auto',
          scrollbarWidth: 'thin',
          '& .MuiToggleButton-root': {
            flex: '1 0 auto',
            whiteSpace: 'nowrap',
            gap: 1,
            border: 0,
            borderRadius: 0,
            py: 1.25,
            textTransform: 'none',
            fontSize: 14,
            color: 'text.primary',
            bgcolor: 'transparent',
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
          <ToggleButton
            key={tab.id}
            value={tab.id}
            data-section={tab.id}
            tabIndex={tab.id === current ? 0 : -1}
          >
            <Box component="span">{tab.label}</Box>
            <Box
              component="span"
              sx={{
                px: 0.75,
                minWidth: 20,
                borderRadius: '10px',
                fontSize: 12,
                lineHeight: '20px',
                bgcolor: 'var(--mortar-overlay-30)',
              }}
            >
              {tab.count}
            </Box>
          </ToggleButton>
        ))}
      </ToggleButtonGroup>
      <Box sx={{ display: 'flex', alignItems: 'center', flexShrink: 0, px: 1 }}>{actions}</Box>
    </Box>
  )
}
