import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import type { Drift } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { localId } from './dependents.ts'
import { type Problem, sameId } from './lookup.ts'
import { useMods } from './store.ts'

type Describe = (p: Problem) => string

function describeRunError(runError: Extract<Problem, { kind: 'runError' }>['runError']): string {
  const { name, first, count, updated } = runError
  if (updated) {
    return first === ''
      ? i18n._(
          msg`${plural(count, {
            one: `${name} was updated since this run and logged # error.`,
            other: `${name} was updated since this run and logged # errors.`,
          })}`,
        )
      : i18n._(
          msg`${plural(count, {
            one: `${name} was updated since this run and logged # error. First: ${first}`,
            other: `${name} was updated since this run and logged # errors. First: ${first}`,
          })}`,
        )
  }
  return first === ''
    ? i18n._(
        msg`${plural(count, {
          one: `${name} logged # error in the last run.`,
          other: `${name} logged # errors in the last run.`,
        })}`,
      )
    : i18n._(
        msg`${plural(count, {
          one: `${name} logged # error in the last run. First: ${first}`,
          other: `${name} logged # errors in the last run. First: ${first}`,
        })}`,
      )
}

function describeBroken(p: Extract<Problem, { kind: 'broken' }>): string {
  const { name, brokeIn, status, summary } = p.broken
  if (status === 'abandoned') {
    return summary
      ? i18n._(msg`${name} is abandoned: ${summary}`)
      : i18n._(msg`${name} is marked abandoned and may no longer be maintained.`)
  }
  if (status === 'obsolete') {
    return summary
      ? i18n._(msg`${name} is obsolete: ${summary}`)
      : i18n._(msg`${name} is marked obsolete and may not work with this game version.`)
  }
  return brokeIn
    ? i18n._(msg`${name} broke in ${brokeIn}.`)
    : i18n._(msg`${name} is marked broken for this game version.`)
}

function describeAsset(p: Extract<Problem, { kind: 'asset' }>): string {
  const { names, kind, winnerName, overridden } = p.asset
  const target = listNames([p.asset.target, ...(p.siblings ?? []).map((s) => s.target)])
  const who = listNames(names ?? [])
  let winner = ''
  if (winnerName === 'unclear') {
    winner = i18n._(msg` The winner is unclear.`)
  } else if (winnerName) {
    const losers = listNames(overridden ?? [])
    winner =
      losers === ''
        ? i18n._(msg` ${winnerName} wins.`)
        : i18n._(msg` ${winnerName} wins; ${losers} overridden.`)
  }
  const sentence =
    kind === 'load'
      ? i18n._(msg`${who} all load ${target}.${winner}`)
      : i18n._(msg`${who} all edit ${target}.${winner}`)
  return p.asset.info
    ? `${sentence} ${i18n._(msg`These mods may add the same item: ${p.asset.info}.`)}`
    : sentence
}

function describeMissing(
  p: Extract<Problem, { kind: 'missing' }>,
  nameOf: (id: string) => string,
): string {
  const { dependentName, minimumVersion, installedVersion, reason } = p.missing
  const dep = p.missing.listed
    ? p.missing.where?.pageName?.trim() || nameOf(p.missing.id)
    : nameOf(p.missing.id)
  if (p.missing.listed) {
    const note = p.missing.note.trim()
    return note === ''
      ? i18n._(msg`${dependentName}'s Nexus page lists ${dep} as a requirement.`)
      : i18n._(msg`${dependentName}'s Nexus page lists ${dep} as a requirement: ${note}`)
  }
  if (reason === 'disabled') {
    return i18n._(msg`${dependentName} needs ${dep}, which is disabled.`)
  }
  if (reason === 'outdated') {
    return i18n._(
      msg`${dependentName} needs ${dep} ${minimumVersion} or newer, and this profile has ${installedVersion}.`,
    )
  }
  return i18n._(msg`${dependentName} needs ${dep}, which this profile lacks.`)
}

function describeSetting(s: Extract<Problem, { kind: 'setting' }>['setting']): string {
  const installed = listNames(s.forNames ?? [])
  if (!s.variant) {
    return i18n._(
      msg`${s.name} has a ${s.field} setting for ${installed}; it is ${s.current}, so those patches are off.`,
    )
  }
  if (s.currentFor !== '' && installed !== '') {
    return i18n._(
      msg`${s.name}'s ${s.field} is ${s.current}, made for ${s.currentFor}, which this profile lacks; it can match ${installed} instead.`,
    )
  }
  if (s.currentFor !== '') {
    return i18n._(
      msg`${s.name}'s ${s.field} is ${s.current}, made for ${s.currentFor}, which this profile lacks.`,
    )
  }
  return i18n._(msg`${s.name}'s ${s.field} is ${s.current}; it can match ${installed} instead.`)
}

function describeDuplicate(d: Extract<Problem, { kind: 'duplicate' }>['duplicate']): string {
  if ((d.nexusFiles ?? []).length > 1) {
    const files = listNames(
      (d.nexusFiles ?? []).map((file) => `${file.fileName} (${file.version})`),
    )
    return d.nexusOptional
      ? i18n._(msg`${d.name} has multiple Nexus files installed: ${files}. One may be an add-on.`)
      : i18n._(msg`${d.name} has multiple Nexus files installed: ${files}.`)
  }
  return i18n._(msg`${d.name} is installed twice, and only one of them loads.`)
}

// The one-line sentence for a problem, shared by the summary rows and the card badges.
export function useDescribe(): Describe {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const nameOf = (id: string) => mods.find((m) => sameId(m.id, id))?.name ?? localId(id)

  return (p: Problem): string => {
    switch (p.kind) {
      case 'duplicate':
        return describeDuplicate(p.duplicate)
      case 'broken':
        return describeBroken(p)
      case 'asset':
        return describeAsset(p)
      case 'missing':
        return describeMissing(p, nameOf)
      case 'runError':
        return describeRunError(p.runError)
      case 'loadFailure': {
        const { plugin, name } = p.loadFailure
        return name !== '' && name !== plugin
          ? t`${plugin} failed to load (${name}).`
          : t`${plugin} failed to load.`
      }
      case 'damaged':
        return t`${p.damaged.name} has damaged files.`
      case 'deprecated': {
        const { name, replacement } = p.deprecated
        return replacement
          ? t`${name} is deprecated on Thunderstore; its page points to ${replacement}.`
          : t`${name} is deprecated on Thunderstore.`
      }
      case 'pluginClash': {
        const names = listNames((p.pluginClash.copies ?? []).map((c) => c.name))
        return t`${names} all ship the plugin ${p.pluginClash.guid}, and BepInEx loads only one.`
      }
      case 'setting':
        return describeSetting(p.setting)
      default:
        return ''
    }
  }
}

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
      return t`${d.folder} was added to this profile's mods folder by hand, not installed by Mortar.`
    }
    const name = entryName(d.key)
    if (d.kind === 'deleted') {
      return t`${name} was removed from this profile's mods folder.`
    }
    return t`${name} was changed outside Mortar.`
  }
}
