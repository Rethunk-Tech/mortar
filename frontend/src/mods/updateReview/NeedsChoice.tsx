import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import type { ComponentProps } from 'react'
import { ReviewList } from './ReviewList.tsx'

/** Updates from another site than the mod came from: Update all leaves them out, each is confirmed on its own. */
export function NeedsChoice(props: ComponentProps<typeof ReviewList>) {
  const { t } = useLingui()
  if (props.list.length === 0) {
    return null
  }
  return (
    <>
      <Typography sx={{ px: 3, pt: 1.5, fontWeight: 600 }}>
        {t`Needs your choice (${props.list.length})`}
      </Typography>
      <Typography sx={{ px: 3, fontSize: 13, color: 'text.secondary' }}>
        {t`These come from a different site than the one you installed them from. Update all leaves them out.`}
      </Typography>
      <ReviewList {...props} />
    </>
  )
}
