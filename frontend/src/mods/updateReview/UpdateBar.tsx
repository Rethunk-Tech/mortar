import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ArrowUp } from 'lucide-react'
import { useProfiles } from '../../profiles/store.ts'
import { useBadges } from '../badges.ts'
import { visibleUpdates } from '../lookup.ts'
import { accent } from '../paper.ts'
import { useUpdates } from '../updates.ts'

export function UpdateBar() {
  const { t } = useLingui()
  const updates = useUpdates((s) => s.updates)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const openId = useProfiles((s) => s.openId)
  const count = useBadges((s) => s.byProfile[openId]?.updates ?? 0)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === openId))
  const [first] = visibleUpdates(updates, profile)
  if (count === 0) {
    return updates?.unknown ? (
      <Typography sx={{ mx: 2, mt: 1, fontSize: 12, color: 'text.secondary' }}>
        {t`Updates are unknown: the update service could not be reached.`}
      </Typography>
    ) : null
  }
  return (
    <Box
      sx={{
        mx: 2,
        mt: 1.25,
        minHeight: 38,
        flexShrink: 0,
        display: 'flex',
        alignItems: 'center',
        gap: 1.25,
        pl: 1.5,
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
        {t`${plural(count, { one: '# update ready', other: '# updates ready' })}`}
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
}
