import { Box, Button, Link, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { space } from '../theme/density.ts'

const STRONG = 600
const PLAIN = 400

export interface SectionAction {
  label: string
  icon?: ReactNode
  onClick: () => void
  disabled?: boolean
}

export interface ProblemTab {
  id: string
  label: string
  count: number
  // Whether the section holds something broken, as against advice such as cleanup or harmless overlaps.
  errors: boolean
  body: ReactNode
  action?: SectionAction
  keep?: boolean
}

// One section: its title and count, its page action if it has one, then its rows.
export function ProblemGroup({ tab }: { tab: ProblemTab }) {
  return (
    <Box
      component="section"
      aria-label={tab.label}
      sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography component="h2" sx={{ fontSize: 16, fontWeight: 600 }}>
          {tab.label}
        </Typography>
        <Typography
          component="span"
          sx={{
            fontSize: 13,
            color: tab.errors ? 'error.main' : 'text.secondary',
            fontWeight: tab.errors ? STRONG : PLAIN,
          }}
        >
          {tab.count}
        </Typography>
        <Box sx={{ flex: 1 }} />
        {tab.action ? (
          <Button
            variant="outlined"
            color="inherit"
            disabled={tab.action.disabled}
            onClick={tab.action.onClick}
            startIcon={tab.action.icon}
            sx={{ height: space.control, borderColor: 'var(--mortar-hairline-20)' }}
          >
            {tab.action.label}
          </Button>
        ) : null}
      </Box>
      {tab.body}
    </Box>
  )
}

// The advice and history sections as links under the problems; the one chosen opens below them.
export function FooterLinks({
  tabs,
  open,
  onToggle,
}: {
  tabs: ProblemTab[]
  open: string | undefined
  onToggle: (id: string) => void
}) {
  return (
    <Box component="nav" sx={{ display: 'flex', flexWrap: 'wrap', columnGap: 2, rowGap: 0.5 }}>
      {tabs.map((tab) => (
        <Link
          key={tab.id}
          component="button"
          underline="hover"
          aria-expanded={open === tab.id}
          onClick={() => onToggle(tab.id)}
          sx={{
            fontSize: 13,
            color: open === tab.id ? 'text.primary' : 'text.secondary',
            fontWeight: open === tab.id ? STRONG : PLAIN,
          }}
        >
          {tab.label}{' '}
          <Box component="b" sx={{ color: 'text.primary', fontWeight: 600 }}>
            {tab.count}
          </Box>
        </Link>
      ))}
    </Box>
  )
}
