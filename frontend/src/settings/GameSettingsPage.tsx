import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Typography } from '@mui/material'
import { ArrowLeft } from 'lucide-react'
import { useEffect } from 'react'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { GameSettings } from './sections/GameSettings.tsx'
import { shouldLeavePageOnEscape } from './shouldLeavePageOnEscape.ts'

export function GameSettingsPage() {
  const { t } = useLingui()
  const name = useProfiles((s) => s.game?.name ?? '')
  const close = useNav((s) => s.closeGameSettings)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (shouldLeavePageOnEscape(e, document.querySelector('[role="dialog"]') !== null)) {
        close()
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [close])
  return (
    <Box sx={{ height: '100%', overflow: 'auto', px: 3.5, pt: 3, pb: 1.5 }}>
      <Box sx={{ maxWidth: 720, display: 'flex', flexDirection: 'column', gap: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <ButtonBase
            aria-label={t`Back to ${name}`}
            onClick={close}
            sx={{
              width: 36,
              height: 36,
              flexShrink: 0,
              borderRadius: '6px',
              '&:hover': { bgcolor: 'action.hover' },
            }}
          >
            <ArrowLeft size={20} />
          </ButtonBase>
          <Typography component="h1" sx={{ fontSize: 22, fontWeight: 700 }}>
            {t`${name} settings`}
          </Typography>
        </Box>
        <GameSettings />
      </Box>
    </Box>
  )
}
