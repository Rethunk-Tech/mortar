import { useLingui } from '@lingui/react/macro'
import { type Problem, sameId } from './lookup.ts'
import { useMods } from './store.ts'

// The one-line sentence for a problem, shared by the summary rows and the card badges.
export function useDescribe() {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const nameOf = (id: string) => mods.find((m) => sameId(m.uniqueId, id))?.name ?? id
  return (p: Problem): string => {
    if (p.kind === 'duplicate') {
      return t`${p.duplicate.name} is installed twice, and SMAPI loads only one.`
    }
    if (p.kind === 'broken') {
      const { name, brokeIn, status } = p.broken
      if (status === 'obsolete') {
        return t`${name} is marked obsolete and may not work with this game version.`
      }
      return brokeIn
        ? t`${name} broke in ${brokeIn}.`
        : t`${name} is marked broken for this game version.`
    }
    if (p.kind === 'asset') {
      const { names, target, kind } = p.asset
      const who = (names ?? []).join(', ')
      if (kind === 'load') {
        return t`${who} all load ${target}; only one wins.`
      }
      return t`${who} all edit ${target}; the result depends on order.`
    }
    const { dependentName, minimumVersion, installedVersion, reason } = p.missing
    const dep = nameOf(p.missing.uniqueId)
    if (reason === 'disabled') {
      return t`${dependentName} needs ${dep}, which is switched off.`
    }
    if (reason === 'outdated') {
      return t`${dependentName} needs ${dep} ${minimumVersion} or newer, and this profile has ${installedVersion}.`
    }
    return t`${dependentName} needs ${dep}, which this profile lacks.`
  }
}
