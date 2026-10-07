import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { Check } from 'lucide-react'
import { space } from '../theme/density.ts'

export function InstallSteps({ steps }: { steps: string[] }) {
  const { t } = useLingui()
  const labels: Record<string, string> = {
    downloaded: t`Downloaded`,
    files: t`Files added`,
    launcher: t`Launcher replaced`,
    bundled: t`Bundled mods added`,
  }
  return (
    <Box sx={{ display: 'flex', gap: space.gap, flexShrink: 0 }}>
      {steps.map((step) => (
        <Box
          key={step}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 0.5,
            fontSize: 13,
            color: 'text.secondary',
            whiteSpace: 'nowrap',
          }}
        >
          <Check size={14} />
          {labels[step] ?? step}
        </Box>
      ))}
    </Box>
  )
}
