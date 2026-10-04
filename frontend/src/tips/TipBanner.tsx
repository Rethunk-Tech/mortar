import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { SetTipsSeen } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { errorText } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
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
        gap: 1,
        px: 1.5,
        py: 1,
        mx: 2,
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
          SetTipsSeen([...(seen ?? []), tip]).catch((err: unknown) => {
            const body = errorText(err)
            useToasts.getState().push({
              kind: 'error',
              title: t`Couldn't save that setting`,
              ...(body ? { body } : {}),
            })
          })
        }}
        sx={{ whiteSpace: 'nowrap' }}
      >
        <X size={14} />
      </TipIconButton>
    </Box>
  )
}
