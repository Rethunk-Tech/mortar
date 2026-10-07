import { useLingui } from '@lingui/react/macro'
import type { GroupKind } from './compare.ts'

export function useGroupTitle(aName: string, bName: string): (kind: GroupKind) => string {
  const { t } = useLingui()
  const titles: Record<GroupKind, string> = {
    version: t`Different version`,
    onlyB: t`Only in ${{ name: bName }}`,
    onlyA: t`Only in ${{ name: aName }}`,
    enabled: t`Enabled in one only`,
    source: t`Different source`,
    identical: t`Identical`,
  }
  return (kind) => titles[kind]
}
