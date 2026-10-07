import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { CloudOff } from 'lucide-react'
import { useEffect } from 'react'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import {
  RECHECK_MS,
  recheck,
  refresh,
  retry,
  STATE_POLL_MS,
  unreachable,
  useOffline,
} from './offline.ts'
import { offlineMessage } from './offlineText.ts'

// Runs only while the window is focused: a source that answered costs nothing, an unreachable one gets one probe
// every five minutes.
function useOfflineSync() {
  useEffect(() => {
    const focused = () => document.hasFocus()
    const run = (task: () => Promise<void>) => {
      if (focused()) {
        task().catch(reportUnexpected)
      }
    }
    run(refresh)
    const poll = globalThis.setInterval(() => run(refresh), STATE_POLL_MS)
    const probe = globalThis.setInterval(() => run(recheck), RECHECK_MS)
    const onFocus = () => run(refresh)
    globalThis.addEventListener('focus', onFocus)
    // Go says the moment a source flips, so the banner does not wait for the poll.
    const off = Events.On('netstate:changed', () => run(refresh))
    return () => {
      globalThis.clearInterval(poll)
      globalThis.clearInterval(probe)
      globalThis.removeEventListener('focus', onFocus)
      off()
    }
  }, [])
}

// One slim row under the title bar while a mod source cannot be reached.
export function OfflineBanner() {
  const { t, i18n } = useLingui()
  useOfflineSync()
  const down = unreachable(useOffline((s) => s.states))
  if (down.length === 0) {
    return null
  }
  return (
    <Box
      role="status"
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        px: space.pad,
        minHeight: 32,
        flexShrink: 0,
        fontSize: 13,
        bgcolor: 'background.paper',
        borderBottom: '1px solid var(--mortar-hairline)',
      }}
    >
      <Box component={CloudOff} size={16} aria-hidden={true} sx={{ color: 'warning.main' }} />
      <Box component="span" sx={{ flex: 1, minWidth: 0 }}>
        {offlineMessage(down, i18n.locale)}
      </Box>
      <Button size="small" onClick={() => retry(down).catch(reportUnexpected)}>
        {t`Retry`}
      </Button>
    </Box>
  )
}
