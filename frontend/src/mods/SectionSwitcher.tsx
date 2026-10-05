import { useLingui } from '@lingui/react/macro'
import { Box, Button, ToggleButton, ToggleButtonGroup } from '@mui/material'
import type { KeyboardEvent } from 'react'
import { type SectionTab, stepSection } from './problemSection.ts'

export interface SectionAction {
  label: string
  onClick: () => void
  disabled?: boolean
}

// The Problems tab's one header row: an exclusive segment per section with its count, and the chosen section's
// action at the end of the same row. Left and Right move between segments.
export function SectionSwitcher({
  tabs,
  current,
  onChoose,
  action,
}: {
  tabs: readonly SectionTab[]
  current: string
  onChoose: (id: string) => void
  action: SectionAction | undefined
}) {
  const { t } = useLingui()
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
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
      <ToggleButtonGroup
        exclusive={true}
        size="small"
        value={current}
        aria-label={t`Problem sections`}
        onKeyDown={onKeyDown}
        onChange={(_e, id: string | null) => {
          if (id !== null) {
            onChoose(id)
          }
        }}
        sx={{ flexWrap: 'wrap' }}
      >
        {tabs.map((tab) => (
          <ToggleButton
            key={tab.id}
            value={tab.id}
            data-section={tab.id}
            tabIndex={tab.id === current ? 0 : -1}
          >
            {`${tab.label} ${tab.count}`}
          </ToggleButton>
        ))}
      </ToggleButtonGroup>
      {action ? (
        <Button
          size="small"
          disabled={action.disabled}
          onClick={action.onClick}
          sx={{ height: 30 }}
        >
          {action.label}
        </Button>
      ) : null}
    </Box>
  )
}
