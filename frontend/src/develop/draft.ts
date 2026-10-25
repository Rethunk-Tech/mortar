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
  private: boolean
  // The Thunderstore packages (Team-Name) the mod starts depending on; null until the player sees the recommendations.
  dependencies: string[] | null
  // The picked packages the mod only uses when present (soft); the rest it requires.
  optional: string[]
}

// The projects folder, and each remembered parameter as "param.<id>".
export type Remembered = Partial<Record<string, string>>

// The same rules as devsvc's NewMod.check, so the button says why before Mortar is asked.
const MOD_NAME = /^[A-Z][A-Za-z0-9]{1,63}$/
const FOLDER = /^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$/
const OWNER = /^[A-Za-z0-9][A-Za-z0-9-]{0,38}$/

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
    private: false,
    dependencies: null,
    optional: [],
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

// The wizard's steps: the mod itself, the author details Mortar remembers, and where the project goes.
export const STEPS = ['mod', 'author', 'dependencies', 'project'] as const
export type Step = (typeof STEPS)[number]

/** The parameters a step asks for: remembered ones on the author step, the rest on the mod step. */
export function stepParams(form: Form | null, step: Step): Param[] {
  const all = form?.parameters ?? []
  if (step === 'project' || step === 'dependencies') {
    return []
  }
  return all.filter((p) => p.remember === (step === 'author'))
}

function firstIssue(d: Draft, params: Param[]): string {
  for (const p of params) {
    const issue = paramIssue(p, d.values[p.id] ?? '')
    if (issue) {
      return issue
    }
  }
  return ''
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
export function stepIssue(d: Draft, form: Form | null, step: Step): string {
  if (!form) {
    return i18n._(msg`Reading the template`)
  }
  if (step === 'mod') {
    if (!MOD_NAME.test(d.name)) {
      return i18n._(
        msg`The mod name starts with a capital letter and holds only letters and digits.`,
      )
    }
    if (!FOLDER.test(d.repo || d.name)) {
      return i18n._(msg`The folder name holds only letters, digits, dots, dashes and underscores.`)
    }
  }
  const issue = firstIssue(d, stepParams(form, step))
  if (issue || step !== 'project') {
    return issue
  }
  if (d.parent === '') {
    return i18n._(msg`Choose the folder for your mod projects.`)
  }
  if (d.github && !OWNER.test(d.owner)) {
    return i18n._(msg`Name the account or organisation on ${{ source: sourceLabel('github') }}.`)
  }
  return ''
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
