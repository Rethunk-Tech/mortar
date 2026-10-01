import { useLingui } from '@lingui/react/macro'
import type { Drift } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { type Problem, sameId } from './lookup.ts'
import { useMods } from './store.ts'

type Describe = (p: Problem) => string

export function useDescribeDrift(): (d: Drift) => string {
  const { t } = useLingui()
  return (d) => {
    if (d.kind === 'unknown') {
      return t`${d.folder} is in this profile's mods folder and is not an installed entry.`
    }
    if (d.kind === 'deleted') {
      return t`${d.key} was removed from this profile's mods folder.`
    }
    return t`${d.key} was changed outside Mortar.`
  }
}

// The one-line sentence for a problem, shared by the summary rows and the card badges.
export function useDescribe(): Describe {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const nameOf = (id: string) => mods.find((m) => sameId(m.uniqueId, id))?.name ?? id

  const describeBroken = (p: Extract<Problem, { kind: 'broken' }>): string => {
    const { name, brokeIn, status, summary } = p.broken
    if (status === 'abandoned') {
      return summary
        ? t`${name} is abandoned: ${summary}`
        : t`${name} is marked abandoned and may no longer be maintained.`
    }
    if (status === 'obsolete') {
      return summary
        ? t`${name} is obsolete: ${summary}`
        : t`${name} is marked obsolete and may not work with this game version.`
    }
    return brokeIn
      ? t`${name} broke in ${brokeIn}.`
      : t`${name} is marked broken for this game version.`
  }

  const describeAsset = (p: Extract<Problem, { kind: 'asset' }>): string => {
    const { names, target, kind, winnerName, overridden } = p.asset
    const who = (names ?? []).join(', ')
    let winner = ''
    if (winnerName === 'unclear') {
      winner = t` The winner is unclear.`
    } else if (winnerName) {
      const losers = (overridden ?? []).join(', ')
      winner =
        losers === '' ? t` ${winnerName} wins.` : t` ${winnerName} wins; ${losers} overridden.`
    }
    if (kind === 'load') {
      return t`${who} all load ${target}.${winner}`
    }
    return t`${who} all edit ${target}.${winner}`
  }

  const describeMissing = (p: Extract<Problem, { kind: 'missing' }>): string => {
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

  return (p: Problem): string => {
    switch (p.kind) {
      case 'duplicate':
        return t`${p.duplicate.name} is installed twice, and SMAPI loads only one.`
      case 'broken':
        return describeBroken(p)
      case 'asset':
        return describeAsset(p)
      case 'missing':
        return describeMissing(p)
      case 'runError': {
        const { name, first, count } = p.runError
        if (first !== '') {
          return t`${name} logged an error in the last run: ${first}`
        }
        return t`${name} logged ${count} errors in the last run.`
      }
      default:
        return ''
    }
  }
}
