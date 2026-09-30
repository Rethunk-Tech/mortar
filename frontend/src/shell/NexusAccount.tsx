import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip } from '@mui/material'
import { LogIn, LogOut } from 'lucide-react'
import { SignOut } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import { openSettings } from '../nav/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function NexusAccount({ onNavigate }: { onNavigate: () => void }) {
  const { t } = useLingui()
  const { signedIn, name, premium } = useNexus()
  return (
    <Box
      sx={{
        mt: 'auto',
        p: 1.5,
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        borderTop: '1px solid rgba(255,255,255,0.12)',
      }}
    >
      <Box sx={{ minWidth: 0, flexGrow: 1, display: 'flex', flexDirection: 'column', gap: '4px' }}>
        <Box sx={{ fontSize: 12, color: 'rgba(225,225,230,0.95)' }}>{t`Nexus Mods`}</Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
          <Box sx={{ fontSize: 14, fontWeight: 600, overflow: 'hidden', textOverflow: 'ellipsis' }}>
            {signedIn ? name : t`Not signed in`}
          </Box>
          {signedIn ? (
            <Chip
              size="small"
              color={premium ? 'primary' : 'default'}
              label={premium ? t`Premium` : t`Free`}
            />
          ) : null}
        </Box>
      </Box>
      {signedIn ? (
        <Button
          size="small"
          startIcon={<LogOut size={16} />}
          onClick={() => {
            SignOut().catch(reportUnexpected)
          }}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Sign out`}
        </Button>
      ) : (
        <Button
          size="small"
          startIcon={<LogIn size={16} />}
          onClick={() => {
            onNavigate()
            openSettings('nexus')
          }}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Sign in`}
        </Button>
      )}
    </Box>
  )
}
