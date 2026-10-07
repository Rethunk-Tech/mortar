import { useLingui } from '@lingui/react/macro'
import { useProfileLoader } from '../profiles/store.ts'
import { type GroupBy, listHeadingFor } from './group.ts'

// The heading of a group in the Mods list, cards and Config list, by the chosen grouping.
export function useListHeading(groupBy: GroupBy) {
  const { t } = useLingui()
  const loaderName = useProfileLoader()?.name ?? ''
  return listHeadingFor(groupBy, {
    category: t`Uncategorised`,
    source: t`Unknown source`,
    tag: t`Untagged`,
    author: t`Unknown author`,
    group: t`Ungrouped`,
    problems: t`Mods with problems`,
    update: t`Update available`,
    enabled: t`Enabled`,
    disabled: t`Disabled`,
    smapi: t`${loaderName} mods`,
  })
}
