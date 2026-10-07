import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import type { ComponentProps } from 'react'
import { space } from '../../theme/density.ts'
import { ReviewList } from './ReviewList.tsx'

/** Updates from another site than the mod came from: Update all leaves them out, each is confirmed on its own. */
export function NeedsChoice(props: ComponentProps<typeof ReviewList>) {
  const { t } = useLingui()
  if (props.list.length === 0) {
    return null
  }
  return (
    <>
      <Typography sx={{ px: space.pad, pt: space.pad, fontWeight: 600 }}>
        {t`Needs your choice (${props.list.length})`}
      </Typography>
      <Typography sx={{ px: space.pad, fontSize: 13, color: 'text.secondary' }}>
        {t`These come from a different site than you installed them from. Update all skips them.`}
      </Typography>
      <ReviewList {...props} />
    </>
  )
}
