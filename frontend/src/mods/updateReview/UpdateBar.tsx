import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { ArrowUp } from 'lucide-react'
import { useProfiles } from '../../profiles/store.ts'
import { updateCount } from '../lookup.ts'
import { useNexusDetails } from '../nexusDetails.ts'
import { accent } from '../paper.ts'
import { useUpdates } from '../updates.ts'

export function UpdateBar() {
  const { t } = useLingui()
  const updates = useUpdates((s) => s.updates)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const count = updateCount(updates, profile, useNexusDetails.getState().byId)
  if (count === 0) {
    return updates?.unknown ? (
      <Typography sx={{ mx: 2, mt: 1, fontSize: 12, color: 'text.secondary' }}>
        {t`Updates are unknown: SMAPI's update service could not be reached.`}
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
      <Box component="span" sx={{ display: 'flex', flexShrink: 0, color: 'primary.main' }}>
        <ArrowUp size={16} aria-hidden={true} />
      </Box>
      <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
        {t`${plural(count, { one: '# update is available', other: '# updates are available' })}`}
      </Typography>
      <Button
        size="small"
        variant="contained"
        onClick={() => setReviewing(true)}
        sx={{ flexShrink: 0 }}
      >
        {t`Review`}
      </Button>
    </Box>
  )
}
