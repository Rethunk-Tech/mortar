import { useLingui } from '@lingui/react/macro'

// Each Redundant kind's reason, naming the enabled mods that make it redundant.
export function useRedundantReason() {
  const { t } = useLingui()
  return (item: {
    kind: string
    by: { name: string }[] | null
    detail?: string
    covered?: boolean
  }) => {
    const names = (item.by ?? []).map((by) => by.name).join(', ')
    if (item.kind === 'superseded') {
      return t`Replaced by ${names}, which is also enabled`
    }
    if (item.kind === 'shadowed') {
      return t`Every edit it makes is overwritten by ${names}`
    }
    const detail = item.detail ?? ''
    if (item.covered) {
      return t`Everything it changes, ${names} also changes: ${detail}`
    }
    return t`Does the same job as ${names}: both change ${detail}`
  }
}
