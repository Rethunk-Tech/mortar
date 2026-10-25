import { msg } from '@lingui/core/macro'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { i18n } from '../i18n/index.ts'

export const REMEMBERED = 'mortar.newMod'

// What the New mod dialog edits; repo empty means the mod's name.
export interface Draft {
  name: string
  description: string
  author: string
  team: string
  guidPrefix: string
  parent: string
  repo: string
  profile: string
  github: boolean
  owner: string
  private: boolean
}

const MAX_DESCRIPTION = 250
// The same rules as devsvc's NewMod.check, so the button says why before Mortar is asked.
const MOD_NAME = /^[A-Z][A-Za-z0-9]{1,63}$/
const TEAM = /^[A-Za-z0-9_]{1,64}$/
const GUID_PREFIX = /^[a-z0-9]+(\.[a-z0-9]+)*$/
const FOLDER = /^[A-Za-z0-9][A-Za-z0-9._-]{0,99}$/
const OWNER = /^[A-Za-z0-9][A-Za-z0-9-]{0,38}$/

export type Remembered = Partial<Record<'author' | 'team' | 'guidPrefix' | 'parent', string>>

/** A fresh draft with the author, team, id prefix and folder the player used last. */
export function firstDraft(remembered: Remembered, owner: string): Draft {
  return {
    name: '',
    description: '',
    author: remembered.author ?? '',
    team: remembered.team ?? '',
    guidPrefix: remembered.guidPrefix ?? '',
    parent: remembered.parent ?? '',
    repo: '',
    profile: 'new',
    github: owner !== '',
    owner,
    private: false,
  }
}

/** The first thing Mortar would refuse in the draft, as the button's reason; empty when it can start. */
export function draftIssue(d: Draft): string {
  const repo = d.repo || d.name
  if (!MOD_NAME.test(d.name)) {
    return i18n._(msg`The mod name starts with a capital letter and holds only letters and digits.`)
  }
  if (d.description.trim() === '' || d.description.length > MAX_DESCRIPTION) {
    return i18n._(
      msg`Describe the mod in one line of at most ${{ max: MAX_DESCRIPTION }} characters.`,
    )
  }
  if (d.author.trim() === '') {
    return i18n._(msg`Name the author.`)
  }
  if (!TEAM.test(d.team)) {
    return i18n._(
      msg`The team on ${{ source: sourceLabel('thunderstore') }} holds only letters, digits and underscores.`,
    )
  }
  if (!GUID_PREFIX.test(d.guidPrefix)) {
    return i18n._(
      msg`The id prefix holds lower-case letters and digits between dots, such as com.yourname.`,
    )
  }
  if (d.parent === '') {
    return i18n._(msg`Choose the folder for your mod projects.`)
  }
  if (!FOLDER.test(repo)) {
    return i18n._(msg`The folder name holds only letters, digits, dots, dashes and underscores.`)
  }
  if (d.github && !OWNER.test(d.owner)) {
    return i18n._(msg`Name the account or organisation on ${{ source: sourceLabel('github') }}.`)
  }
  return ''
}
