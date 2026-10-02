import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
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
  const daily = `${limits.daily.remaining} of ${limits.daily.limit}`
  const hourly = `${limits.hourly.remaining} of ${limits.hourly.limit}`
  const throttled =
    limits.daily.remaining <= remainingFloor || limits.hourly.remaining <= remainingFloor
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px', fontSize: 14 }}>
      <Box>{t`${daily} requests left today · ${hourly} this hour`}</Box>
      {throttled ? (
        <Box sx={{ color: 'rgba(225,225,230,0.95)' }}>
          {t`Nexus is throttling requests until the window resets.`}
        </Box>
      ) : null}
    </Box>
  )
}
