import { useLingui } from '@lingui/react/macro'
import type { Drift } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { type Problem, sameId } from './lookup.ts'
import { useMods } from './store.ts'

type Describe = (p: Problem) => string

export function useDescribeDrift(): (d: Drift) => string {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  // A drift names an archive entry; the user knows it by the mods it installed.
  const entryName = (key: string) =>
    mods
      .filter((m) => m.key === key)
      .map((m) => m.name)
      .join(', ') || key
  return (d) => {
    if (d.kind === 'unknown') {
      return t`${d.folder} is in this profile's mods folder and is not an installed entry.`
    }
    const name = entryName(d.key)
    if (d.kind === 'deleted') {
      return t`${name} was removed from this profile's mods folder.`
    }
    return t`${name} was changed outside Mortar.`
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
    const sentence =
      kind === 'load'
        ? t`${who} all load ${target}.${winner}`
        : t`${who} all edit ${target}.${winner}`
    return p.asset.info
      ? `${sentence} ${t`These mods may add the same item: ${p.asset.info}.`}`
      : sentence
  }

  const describeMissing = (p: Extract<Problem, { kind: 'missing' }>): string => {
    const { dependentName, minimumVersion, installedVersion, reason } = p.missing
    const dep = p.missing.listed
      ? p.missing.where?.pageName?.trim() || nameOf(p.missing.uniqueId)
      : nameOf(p.missing.uniqueId)
    if (p.missing.listed) {
      const note = p.missing.note.trim()
      return note === ''
        ? t`${dependentName}'s Nexus page lists ${dep} as a requirement.`
        : t`${dependentName}'s Nexus page lists ${dep} as a requirement: ${note}`
    }
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
        if ((p.duplicate.nexusFiles ?? []).length > 1) {
          const files = (p.duplicate.nexusFiles ?? [])
            .map((file) => `${file.fileName} (${file.version})`)
            .join(', ')
          return p.duplicate.nexusOptional
            ? t`${p.duplicate.name} has multiple Nexus files installed: ${files}. One may be an add-on.`
            : t`${p.duplicate.name} has multiple Nexus files installed: ${files}.`
        }
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
      case 'setting': {
        const s = p.setting
        const installed = (s.forNames ?? []).join(', ')
        if (!s.variant) {
          return t`${s.name} has a ${s.field} setting for ${installed}; it is ${s.current}, so those patches are off.`
        }
        if (s.currentFor !== '' && installed !== '') {
          return t`${s.name}'s ${s.field} is ${s.current}, made for ${s.currentFor}, which this profile lacks; it can match ${installed} instead.`
        }
        if (s.currentFor !== '') {
          return t`${s.name}'s ${s.field} is ${s.current}, made for ${s.currentFor}, which this profile lacks.`
        }
        return t`${s.name}'s ${s.field} is ${s.current}; it can match ${installed} instead.`
      }
      default:
        return ''
    }
  }
}
