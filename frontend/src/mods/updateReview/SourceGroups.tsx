import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import type { ComponentProps } from 'react'
import { space } from '../../theme/density.ts'
import { groupBySource } from './groups.ts'
import { ReviewList } from './ReviewList.tsx'

/** The same-source updates, one headed list per reporting source. */
export function SourceGroups(props: ComponentProps<typeof ReviewList>) {
  const { t } = useLingui()
  return groupBySource(props.list).map((g) => {
    const name = g.name || t`Other`
    return (
      <div key={g.name}>
        <Typography sx={{ px: space.pad, pt: space.pad, fontWeight: 600 }}>
          {t`${name} (${g.list.length})`}
        </Typography>
        <ReviewList {...props} list={g.list} />
      </div>
    )
  })
}
