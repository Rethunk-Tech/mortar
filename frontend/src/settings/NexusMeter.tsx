import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { lowerFirst } from '../i18n/lowerFirst.ts'
import { useNexus } from './nexus.ts'

const remainingFloor = 5

export function NexusMeter() {
  const { t } = useLingui()
  const limits = useNexus((s) => s.limits)
  if (!limits.known) {
    return (
      <Box sx={{ fontSize: 13, color: 'text.secondary' }}>
        {t`Request counts appear after Mortar talks to ${{ source: sourceLabel('nexus') }}.`}
      </Box>
    )
  }
  const throttled =
    limits.daily.remaining <= remainingFloor || limits.hourly.remaining <= remainingFloor
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px', fontSize: 14 }}>
      <Box>
        {[
          t`${{ remaining: limits.daily.remaining }} of ${{ limit: limits.daily.limit }} requests left today`,
          t`${{ remaining: limits.hourly.remaining }} of ${{ limit: limits.hourly.limit }} this hour`,
        ].join(' · ')}
      </Box>
      {limits.hourly.reset && limits.daily.reset ? (
        <Box sx={{ fontSize: 13, color: 'text.secondary' }}>
          {[
            t`Hourly count resets ${{ when: formatWhen(limits.hourly.reset) }}`,
            lowerFirst(t`Daily count resets ${{ when: formatWhen(limits.daily.reset) }}`),
          ].join(' · ')}
        </Box>
      ) : null}
      {throttled ? (
        <Box sx={{ color: 'var(--mortar-ink-sec)' }}>
          {t`${{ source: sourceLabel('nexus') }} is throttling requests until the window resets.`}
        </Box>
      ) : null}
    </Box>
  )
}
