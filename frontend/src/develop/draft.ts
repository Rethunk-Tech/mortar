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
    parent: remembered['parent'] ?? '',
    repo: '',
    profile: 'new',
    github: false,
    owner: '',
    private: false,
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
  const name = p.description || p.id
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

/** The first thing Mortar would refuse in the draft, as the button's reason; empty when it can start. */
export function draftIssue(d: Draft, form: Form | null): string {
  if (!form) {
    return i18n._(msg`Reading the template`)
  }
  if (!MOD_NAME.test(d.name)) {
    return i18n._(msg`The mod name starts with a capital letter and holds only letters and digits.`)
  }
  for (const p of form.parameters ?? []) {
    const issue = paramIssue(p, d.values[p.id] ?? '')
    if (issue) {
      return issue
    }
  }
  if (d.parent === '') {
    return i18n._(msg`Choose the folder for your mod projects.`)
  }
  if (!FOLDER.test(d.repo || d.name)) {
    return i18n._(msg`The folder name holds only letters, digits, dots, dashes and underscores.`)
  }
  if (d.github && !OWNER.test(d.owner)) {
    return i18n._(msg`Name the account or organisation on ${{ source: sourceLabel('github') }}.`)
  }
  return ''
}
