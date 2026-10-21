import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ArrowUp } from 'lucide-react'
import { memo, useState } from 'react'
import { useProfiles } from '../../profiles/store.ts'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useBadges } from '../badges.ts'
import { ListCallout } from '../ListCallout.tsx'
import { visibleUpdates } from '../lookup.ts'
import { accent } from '../paper.ts'
import { useUpdates } from '../updates.ts'

// Takes no props, so a re-render of the tab around it skips it.
export const UpdateBar = memo(function UpdateBarView() {
  const { t } = useLingui()
  const updates = useUpdates((s) => s.updates)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const openId = useProfiles((s) => s.openId)
  const count = useBadges((s) => s.byProfile[openId]?.updates ?? 0)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === openId))
  const checkNow = useUpdates((s) => s.checkNow)
  const [first] = visibleUpdates(updates, profile)
  // Dismissed for this answer only: the next check's answer brings the notice back if it is still incomplete.
  const [dismissed, setDismissed] = useState<unknown>(null)
  const [checking, setChecking] = useState(false)
  if (count === 0) {
    if (!updates?.unknown || dismissed === updates) {
      return null
    }
    const retry = () => {
      setChecking(true)
      checkNow()
        .catch(reportUnexpected)
        .finally(() => setChecking(false))
    }
    return (
      <ListCallout
        text={t`Some update checks did not answer in time, so updates for those mods are unknown.`}
        actions={
          <>
            <Button size="small" variant="outlined" disabled={checking} onClick={retry}>
              {checking ? t`Checking…` : t`Check again`}
            </Button>
            <Button size="small" color="inherit" onClick={() => setDismissed(updates)}>
              {t`Dismiss`}
            </Button>
          </>
        }
      />
    )
  }
  return (
    <Box
      sx={{
        mx: space.gutter,
        mt: 1.25,
        minHeight: 38,
        flexShrink: 0,
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        pl: space.pad,
        pr: 0.75,
        fontSize: 14,
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: accent.line,
        borderRadius: '6px',
      }}
    >
      <Box
        component="span"
        sx={{ display: 'flex', flexShrink: 0, color: 'var(--mortar-accent-ink)' }}
      >
        <ArrowUp size={16} aria-hidden={true} />
      </Box>
      <Typography
        component="span"
        noWrap={true}
        sx={{ flexShrink: 0, fontSize: 14, fontWeight: 600 }}
      >
        {plural(count, { one: '# update ready', other: '# updates ready' })}
      </Typography>
      <Typography
        component="span"
        noWrap={true}
        sx={{ flex: 1, minWidth: 0, fontSize: 13, color: 'text.secondary' }}
      >
        {first ? t`including ${first.name} ${first.version}` : ''}
      </Typography>
      <Button
        size="small"
        variant="contained"
        onClick={() => setReviewing(true)}
        sx={{ flexShrink: 0 }}
      >
        {t`Review updates`}
      </Button>
    </Box>
  )
})
