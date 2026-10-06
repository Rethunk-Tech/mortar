import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNexus } from './nexus.ts'

const remainingFloor = 5

export function NexusMeter() {
  const { t } = useLingui()
  const limits = useNexus((s) => s.limits)
  if (!limits.known) {
    return (
      <Box sx={{ fontSize: 13, color: 'text.secondary' }}>
        {t`Request counts appear after Mortar talks to Nexus.`}
      </Box>
    )
  }
  const throttled =
    limits.daily.remaining <= remainingFloor || limits.hourly.remaining <= remainingFloor
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px', fontSize: 14 }}>
      <Box>
        {t`${limits.daily.remaining} of ${limits.daily.limit} requests left today · ${limits.hourly.remaining} of ${limits.hourly.limit} this hour`}
      </Box>
      {limits.hourly.reset && limits.daily.reset ? (
        <Box sx={{ fontSize: 13, color: 'text.secondary' }}>
          {t`Hourly count resets ${formatWhen(limits.hourly.reset)} · daily count resets ${formatWhen(limits.daily.reset)}`}
        </Box>
      ) : null}
      {throttled ? (
        <Box sx={{ color: 'var(--mortar-ink-sec)' }}>
          {t`Nexus is throttling requests until the window resets.`}
        </Box>
      ) : null}
    </Box>
  )
}
