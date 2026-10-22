import { msg, plural } from '@lingui/core/macro'
import { HARMONY, WINDOWS } from '../brand/productNames.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import type { Problem } from './lookup.ts'

type LoadFailureRow = Extract<Problem, { kind: 'loadFailure' }>['loadFailure']

type RanPatch = NonNullable<LoadFailureRow['patch']>

// The last two parts of a "Namespace.Type.Method" name, which is what a player can match to a mod's code.
function shortName(name: string): string {
  return name.split('.').slice(-2).join('.')
}

function patchKindText(kind: string): string {
  switch (kind) {
    case 'prefix':
    case 'prefix-skip':
      return i18n._(msg`prefix`)
    case 'postfix':
      return i18n._(msg`postfix`)
    case 'transpiler':
      return i18n._(msg`transpiler`)
    case 'finalizer':
      return i18n._(msg`finalizer`)
    case 'hook':
      return i18n._(msg`hook`)
    default:
      return i18n._(msg`${{ app: HARMONY }} patch`)
  }
}

// Where the mod's frame is in its code: the source line, or why the log gave none. A module no enabled package ships
// says nothing.
function ranLocation(mod: string, source: string, noSource: string): string {
  const colon = source.lastIndexOf(':')
  if (colon > 0) {
    const file = source.slice(0, colon)
    const line = source.slice(colon + 1)
    return i18n._(msg`, at ${file} line ${line}.`)
  }
  switch (noSource) {
    case 'no-symbols':
      return i18n._(msg`; ${{ name: mod }} ships no debug symbols, so its source line is unknown.`)
    case 'windows-pdb':
      return i18n._(
        msg`; ${{ name: mod }} ships ${{ os: WINDOWS }}-format symbols, which the game cannot read.`,
      )
    case 'no-line':
      return i18n._(msg`; the runtime printed no source line for that method (generated code).`)
    default:
      return '.'
  }
}

// The patch that threw the error, or that called the code that threw it.
function ranPatch(who: string, thrown: string, patch: RanPatch): string {
  const { target, mod = '' } = patch
  const kind = patchKindText(patch.kind ?? '')
  const method = shortName(patch.method)
  const own = mod === '' || mod === who
  if (thrown === patch.method) {
    return own
      ? i18n._(msg`Its ${kind} ${method} on ${target} threw it`)
      : i18n._(msg`${{ name: mod }}'s ${kind} ${method} on ${target} threw it`)
  }
  return own
    ? i18n._(msg`It was thrown in ${thrown}, called from its ${kind} ${method} on ${target}`)
    : i18n._(
        msg`It was thrown in ${thrown}, called from ${{ name: mod }}'s ${kind} ${method} on ${target}`,
      )
}

// Where a ran error was thrown, and the other mods that patch a method on its stack: those are suspects too.
function ranWhere(who: string, f: LoadFailureRow): string {
  const thrown = f.thrown ?? ''
  const patchedBy = f.patchedBy ?? []
  if (thrown === '') {
    return ''
  }
  const mods = listNames(patchedBy)
  const suspects = plural(
    { count: patchedBy.length },
    {
      one: `${mods} also patches code on that path.`,
      other: `${mods} also patch code on that path.`,
    },
  )
  const source = f.source ?? ''
  const noSource = f.noSource ?? ''
  if (
    !f.patch &&
    source === '' &&
    (noSource === '' || noSource === 'not-in-profile') &&
    patchedBy.length > 0
  ) {
    return i18n._(msg`It was thrown in ${thrown}; ${suspects}`)
  }
  const where = f.patch ? ranPatch(who, thrown, f.patch) : i18n._(msg`It was thrown in ${thrown}`)
  const owner = f.patch?.mod || who
  const sentence = where + ranLocation(owner, source, noSource)
  return patchedBy.length === 0 ? sentence : `${sentence} ${suspects}`
}

// A game-setting row's message is the catalog's own wording for the setting that is off.
export function describeLoadFailure(f: LoadFailureRow): string {
  const { plugin, name, kind, dependency, message, ran, errors } = f
  if (ran) {
    const who = name === '' ? plugin : name
    const logged = plural(
      { count: errors ?? 1 },
      {
        one: `${who} loaded and logged an error while running.`,
        other: `${who} loaded and logged # errors while running.`,
      },
    )
    return [logged, ranWhere(who, f)].filter(Boolean).join(' ')
  }
  if (kind === 'game-setting') {
    return message
  }
  if (kind === 'run-failed') {
    return i18n._(msg`The last launch failed: ${message}.`)
  }
  if (kind === 'incompatible-plugin' && dependency) {
    return i18n._(
      msg`${plugin} declares itself incompatible with ${dependency}, which is also enabled.`,
    )
  }
  return name !== '' && name !== plugin
    ? i18n._(msg`${plugin} failed to load (${name}).`)
    : i18n._(msg`${plugin} failed to load.`)
}
