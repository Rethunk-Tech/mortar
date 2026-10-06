import { Box, Chip } from '@mui/material'
import { safeUrl } from './bbcode.ts'
import type { Dependency } from './dependencies.ts'
import { openPage } from './menu.ts'

function DependencyChips({ deps }: { deps: Dependency[] }) {
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
      {deps.map((d) => {
        const url = d.url ? safeUrl(d.url) : undefined
        return (
          <Chip
            key={`${d.name}|${d.url ?? ''}`}
            size="small"
            variant="outlined"
            label={d.name}
            title={d.note || undefined}
            {...(url ? { clickable: true, onClick: () => openPage(url) } : {})}
          />
        )
      })}
    </Box>
  )
}

export { DependencyChips }
