import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { SetTipsSeen } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { type TipId, tipVisible } from './visible.ts'

export function TipBanner({ tip, children }: { tip: TipId; children: ReactNode }) {
  const { t } = useLingui()
  const seen = useSettings((s) => s.tipsSeen)
  if (!tipVisible(seen, tip)) {
    return null
  }
  return (
    <Box
      role="note"
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: space.gap,
        px: space.pad,
        py: space.gap,
        mx: space.gutter,
        mt: 1.25,
        bgcolor: 'var(--mortar-panel-solid)',
        border: '1px solid var(--mortar-hairline-12)',
        borderRadius: '8px',
      }}
    >
      <Typography sx={{ flex: 1, minWidth: 0, fontSize: 13, lineHeight: 1.5 }}>
        {children}
      </Typography>
      <TipIconButton
        label={t`Dismiss`}
        onClick={() => {
          SetTipsSeen([...(seen ?? []), tip]).catch(reportError(t`Could not save that setting`))
        }}
        sx={{ whiteSpace: 'nowrap' }}
      >
        <X size={14} />
      </TipIconButton>
    </Box>
  )
}
