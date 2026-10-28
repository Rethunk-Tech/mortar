import { msg } from '@lingui/core/macro'
import type {
  Form,
  Param,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { i18n } from '../i18n/index.ts'

export const REMEMBERED = 'mortar.newMod'

// What the New mod dialog edits: the mod's name, the template's parameters by id (a multi-choice joined with "|"),
// and where the project goes; repo empty means the mod's name.
export interface Draft {
  name: string
  values: Record<string, string>
  parent: string
  repo: string
  profile: string
  github: boolean
  owner: string
  // Why GitHub would refuse owner/repo (no such account, the name taken), or that the check is running; empty when fine.
  ownerIssue: string
  private: boolean
  // The Thunderstore packages (Team-Name) the mod starts depending on; null until the player sees the recommendations.
  dependencies: string[] | null
  // The picked packages the mod only uses when present (soft); the rest it requires.
  optional: string[]
  // A new development profile starts with the open profile's mods.
  copyMods: boolean
  // One of devsvc.Licenses replacing the template's LICENSE; empty keeps it.
  license: string
  // One of the game's asset kinds for a mod made only of files; empty is a code mod from the template.
  kind: string
}

// The projects folder, and each remembered parameter as "param.<id>".
export type Remembered = Partial<Record<string, string>>

export const isRemembered = (v: unknown): v is Remembered =>
  typeof v === 'object' && v !== null && Object.values(v).every((x) => typeof x === 'string')

// The same rules as devsvc's NewMod.check, so the button says why before Mortar is asked.
const MOD_NAME = /^[A-Z][A-Za-z0-9]{1,63}$/
const FOLDER = /^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$/
export const OWNER = /^[A-Za-z0-9][A-Za-z0-9-]{0,38}$/

const rememberKey = (id: string) => `param.${id}`

/** A fresh draft: each parameter at the value the player gave last (when the template remembers it) or its default. */
export function firstDraft(form: Form | null, remembered: Remembered): Draft {
  const values: Record<string, string> = {}
  for (const p of form?.parameters ?? []) {
    values[p.id] = (p.remember ? remembered[rememberKey(p.id)] : undefined) ?? p.default
  }
  return {
    name: '',
    values,
    parent: remembered.parent ?? '',
    repo: '',
    profile: 'new',
    github: false,
    owner: '',
    ownerIssue: '',
    private: false,
    dependencies: null,
    optional: (form?.recommends ?? []).filter((r) => r.optional).map((r) => r.package),
    copyMods: false,
    license: '',
    kind: '',
  }
}

/** What to remember from a draft that started a mod. */
export function rememberedOf(d: Draft, form: Form): Remembered {
  const out: Remembered = { parent: d.parent }
  for (const p of form.parameters ?? []) {
    if (p.remember) {
      out[rememberKey(p.id)] = d.values[p.id] ?? ''
    }
  }
  return out
}

function paramIssue(p: Param, raw: string): string {
  const v = raw.trim()
  const name = p.label
  if (v === '') {
    return p.required ? i18n._(msg`Fill in: ${{ name }}`) : ''
  }
  if (v.startsWith('-')) {
    return i18n._(msg`${{ name }}: cannot start with a dash.`)
  }
  if (p.maxLength > 0 && v.length > p.maxLength) {
    return i18n._(msg`${{ name }}: at most ${{ max: p.maxLength }} characters.`)
  }
  if (p.pattern && !new RegExp(p.pattern).test(v)) {
    return i18n._(msg`${{ name }}: not in the form the template needs.`)
  }
  return ''
}

// The wizard's steps: the kind of mod, the mod itself, the author details Mortar remembers, what the mod does, its
// dependencies, and where the project goes.
const STEPS = ['kind', 'mod', 'author', 'features', 'dependencies', 'project'] as const
export type Step = (typeof STEPS)[number]

/** Where a step sits among the steps shown now; one that is gone (the template had no features) falls to the next. */
export function stepIndex(steps: Step[], step: Step | null): number {
  if (!step) {
    return 0
  }
  const from = STEPS.indexOf(step)
  const i = steps.findIndex((s) => STEPS.indexOf(s) >= from)
  return i < 0 ? steps.length - 1 : i
}

/** The steps this wizard shows: the kind only when the game offers asset kinds, the features only when the form asks
 * any. */
export function wizardSteps(hasKinds: boolean, form: Form | null): Step[] {
  return STEPS.filter(
    (s) =>
      (s !== 'kind' || hasKinds) && (s !== 'features' || !form || stepParams(form, s).length > 0),
  )
}

const isFeature = (p: Param) => p.kind === 'bool' || (p.kind === 'choice' && !p.multiple)

/** The parameters a step asks for: remembered ones on the author step, switches and single choices on the features
 * step, the rest on the mod step. */
export function stepParams(form: Form | null, step: Step): Param[] {
  const all = (form?.parameters ?? []).filter((p) => !p.remember)
  switch (step) {
    case 'author':
      return (form?.parameters ?? []).filter((p) => p.remember)
    case 'features':
      return all.filter(isFeature)
    case 'mod':
      return all.filter((p) => !isFeature(p))
    default:
      return []
  }
}

function firstIssue(d: Draft, params: Param[], all: Param[]): string {
  for (const p of params) {
    const issue = paramIssue(p, d.values[p.id] ?? '') || unmetRequirement(p, d.values, all)
    if (issue) {
      return issue
    }
  }
  return ''
}

/** Why a parameter cannot be on with the other answers as they are, the template's default standing in for an
 * unanswered one; empty when it can. */
export function requirementIssue(p: Param, values: Record<string, string>, all: Param[]): string {
  for (const [id, want] of Object.entries(p.requires ?? {})) {
    const other = all.find((q) => q.id === id)
    if (!other || want === undefined) {
      continue
    }
    if ((values[id] ?? other.default) !== want) {
      const choice = other.choices?.find((c) => c.value === want)?.label || want
      return i18n._(msg`${{ name: p.label }} needs ${{ other: other.label }}: ${{ choice }}.`)
    }
  }
  return ''
}

function unmetRequirement(p: Param, values: Record<string, string>, all: Param[]): string {
  const v = (values[p.id] ?? '').trim()
  return v === '' || v === 'false' ? '' : requirementIssue(p, values, all)
}

/** A package the template's catalog entry recommends, and the answer that brought it in (empty when it always fits). */
export interface Recommended {
  package: string
  param: string
  value: string
}

/** The recommended packages that fit the draft's answers, each with the first answer that matched it. */
export function recommendations(form: Form | null, values: Record<string, string>): Recommended[] {
  const out: Recommended[] = []
  for (const r of form?.recommends ?? []) {
    if (out.some((o) => o.package === r.package)) {
      continue
    }
    if (!r.param) {
      out.push({ package: r.package, param: '', value: '' })
      continue
    }
    const answered = (values[r.param] ?? '').split('|')
    const value = (r.values ?? []).find((v) => answered.includes(v))
    if (value !== undefined) {
      out.push({ package: r.package, param: r.param, value })
    }
  }
  return out
}

/** The packages the mod will start with: the player's picks, else the recommendations. */
export function chosenDependencies(d: Draft, form: Form | null): string[] {
  return d.dependencies ?? recommendations(form, d.values).map((r) => r.package)
}

/** The first thing that keeps a step from going on, as the button's reason; empty when it can. */
export const validModName = (name: string) => MOD_NAME.test(name)

export function stepIssue(d: Draft, form: Form | null, step: Step): string {
  if (step === 'kind') {
    return ''
  }
  if (!form) {
    return i18n._(msg`Reading the template`)
  }
  if (step === 'mod') {
    // The field itself says what a valid name looks like.
    if (d.name === '') {
      return i18n._(msg`Name the mod.`)
    }
    if (!MOD_NAME.test(d.name)) {
      return i18n._(msg`Fix the mod name.`)
    }
    if (!FOLDER.test(d.repo || d.name)) {
      return i18n._(msg`The folder name holds only letters, digits, dots, dashes and underscores.`)
    }
  }
  const issue = firstIssue(d, stepParams(form, step), form.parameters ?? [])
  if (issue || step !== 'project') {
    return issue
  }
  if (d.parent === '') {
    return i18n._(msg`Choose the folder for your mod projects.`)
  }
  if (d.github && !OWNER.test(d.owner)) {
    return i18n._(msg`Name the account or organisation on ${{ source: sourceLabel('github') }}.`)
  }
  return d.github ? d.ownerIssue : ''
}

/** The first thing Mortar would refuse anywhere in the draft; empty when it can start. */
export function draftIssue(d: Draft, form: Form | null): string {
  for (const step of STEPS) {
    const issue = stepIssue(d, form, step)
    if (issue) {
      return issue
    }
  }
  return ''
}
