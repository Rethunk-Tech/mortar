import { expect, test } from 'bun:test'
import type { Form } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  chosenDependencies,
  draftIssue,
  firstDraft,
  recommendations,
  rememberedOf,
  requirementIssue,
  stepIssue,
  stepParams,
} from './draft.ts'

const param = {
  label: 'Param',
  kind: 'string',
  default: '',
  required: true,
  multiple: false,
  choices: [],
  remember: false,
  pattern: '',
  maxLength: 0,
  requires: null,
}
const form: Form = {
  name: 'Test',
  sdk: { version: '', rollForward: '' },
  recommends: [
    { package: 'Evaisa-LethalLib', param: 'categories', values: ['items', 'monsters'] },
    { package: 'AinaVT-LethalConfig', param: 'config', values: ['true'], optional: true },
    { package: 'Always-There', param: '', values: [] },
  ],
  parameters: [
    { ...param, id: 'description', description: 'Description', maxLength: 20 },
    { ...param, id: 'team', description: 'Team', remember: true, pattern: '^[A-Za-z0-9_]+$' },
    {
      ...param,
      id: 'categories',
      description: 'Categories',
      kind: 'choice',
      required: false,
      multiple: true,
      default: 'mods',
    },
    { ...param, id: 'config', description: 'Has settings', kind: 'bool', default: 'false' },
  ],
}

test('a remembered parameter and the projects folder come back; others start at their default', () => {
  const d = firstDraft(form, {
    parent: '/mods',
    'param.team': 'Ab',
    'param.description': 'not remembered',
  })
  expect(d.values).toEqual({ description: '', team: 'Ab', categories: 'mods', config: 'false' })
  expect(d.optional).toEqual(['AinaVT-LethalConfig'])
  expect(d.parent).toBe('/mods')
  expect(rememberedOf({ ...d, values: { ...d.values, team: 'Cd' } }, form)).toEqual({
    parent: '/mods',
    'param.team': 'Cd',
  })
})

test('the template rules are checked before Mortar is asked', () => {
  const ready = { ...firstDraft(form, { parent: '/mods', 'param.team': 'Ab' }), name: 'ShopDrop' }
  expect(draftIssue(ready, form)).not.toBe('')
  const filled = { ...ready, values: { ...ready.values, description: 'Lands.' } }
  expect(draftIssue(filled, form)).toBe('')
  expect(draftIssue({ ...filled, values: { ...filled.values, team: 'a b' } }, form)).not.toBe('')
  expect(
    draftIssue({ ...filled, values: { ...filled.values, description: 'x'.repeat(21) } }, form),
  ).not.toBe('')
  expect(draftIssue({ ...filled, name: 'shop drop' }, form)).not.toBe('')
  expect(draftIssue({ ...filled, github: true, owner: '--help' }, form)).not.toBe('')
  expect(draftIssue({ ...filled, github: true, owner: 'Ab', ownerIssue: 'taken' }, form)).toBe(
    'taken',
  )
  expect(draftIssue(filled, null)).not.toBe('')
})

test('remembered parameters are the author step, switches the features step; each step checks only its own', () => {
  expect(stepParams(form, 'author').map((p) => p.id)).toEqual(['team'])
  expect(stepParams(form, 'mod').map((p) => p.id)).toEqual(['description', 'categories'])
  expect(stepParams(form, 'features').map((p) => p.id)).toEqual(['config'])
  const d = {
    ...firstDraft(form, {}),
    name: 'ShopDrop',
    values: { description: 'Lands.', team: '', categories: 'mods' },
  }
  expect(stepIssue(d, form, 'mod')).toBe('')
  expect(stepIssue(d, form, 'author')).not.toBe('')
  expect(stepIssue(d, form, 'project')).not.toBe('')
})

test('recommended dependencies follow the answers until the player picks their own', () => {
  const d = firstDraft(form, {})
  expect(recommendations(form, d.values).map((r) => r.package)).toEqual(['Always-There'])
  const picked = { ...d, values: { ...d.values, categories: 'mods|monsters', config: 'true' } }
  expect(recommendations(form, picked.values)).toEqual([
    { package: 'Evaisa-LethalLib', param: 'categories', value: 'monsters' },
    { package: 'AinaVT-LethalConfig', param: 'config', value: 'true' },
    { package: 'Always-There', param: '', value: '' },
  ])
  expect(chosenDependencies({ ...picked, dependencies: ['Other-Pick'] }, form)).toEqual([
    'Other-Pick',
  ])
})

test('a switch that needs another answer says so until that answer is given', () => {
  const audience = {
    ...param,
    id: 'audience',
    label: 'Audience',
    description: '',
    kind: 'choice',
    default: 'everyone',
    choices: [
      { value: 'everyone', label: 'Everyone' },
      { value: 'host', label: 'Host only' },
    ],
  }
  const netcode = {
    ...param,
    id: 'netcode',
    label: 'Netcode',
    description: '',
    kind: 'bool',
    requires: { audience: 'everyone' },
  }
  const all = [audience, netcode]
  // The template's default answers for an audience never asked.
  expect(requirementIssue(netcode, {}, all)).toBe('')
  expect(requirementIssue(netcode, { audience: 'host' }, all)).toContain('Everyone')
})
