import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import type { ComponentProps } from 'react'
import { space } from '../../theme/density.ts'
import { useMods } from '../store.ts'
import { splitBySource } from './groups.ts'
import { ReviewList } from './ReviewList.tsx'

/**
 * The same-source updates, one list per reporting source, then the updates of implicit mods as Libraries, each headed
 * when there is more than one.
 */
export function SourceGroups(props: ComponentProps<typeof ReviewList>) {
  const { t } = useLingui()
  const all = useMods((s) => s.mods)
  const { groups, libraries } = splitBySource(props.list, all, props.profile)
  // One section needs no heading: the dialog's title already counts its updates.
  const headed = groups.length + (libraries.length > 0 ? 1 : 0) > 1
  const heading = (text: string) =>
    headed ? (
      <Typography sx={{ px: space.pad, pt: space.pad, fontWeight: 600 }}>{text}</Typography>
    ) : null
  return (
    <>
      {groups.map((g) => {
        const name = g.name || t`Other`
        return (
          <div key={g.name}>
            {heading(t`${name} (${{ count: g.list.length }})`)}
            <ReviewList {...props} list={g.list} />
          </div>
        )
      })}
      {libraries.length > 0 ? (
        <div>
          {heading(t`Libraries (${{ count: libraries.length }})`)}
          <ReviewList {...props} list={libraries} />
        </div>
      ) : null}
    </>
  )
}
