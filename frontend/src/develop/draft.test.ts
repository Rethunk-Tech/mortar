import { expect, test } from 'bun:test'
import type { Form } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { draftIssue, firstDraft, rememberedOf } from './draft.ts'

const param = {
  kind: 'string',
  default: '',
  required: true,
  multiple: false,
  choices: [],
  remember: false,
  pattern: '',
  maxLength: 0,
}
const form: Form = {
  name: 'Test',
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
  ],
}

test('a remembered parameter and the projects folder come back; others start at their default', () => {
  const d = firstDraft(form, {
    parent: '/mods',
    'param.team': 'Ab',
    'param.description': 'not remembered',
  })
  expect(d.values).toEqual({ description: '', team: 'Ab', categories: 'mods' })
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
  expect(draftIssue(filled, null)).not.toBe('')
})
