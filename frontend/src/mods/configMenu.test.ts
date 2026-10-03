import { describe, expect, test } from 'bun:test'
import { draftMap, type GmcmCapture, pendingEdits, valuesEqual } from './configMenu.ts'
import { editableKind, menuControl } from './configMenuKinds.ts'

const capture: GmcmCapture = {
  schema: 1,
  mod: { id: 'A.B', name: 'A', version: '1' },
  gmcmVersion: '1',
  capturedAt: '',
  titleScreenOnlyDefault: false,
  pages: [
    {
      id: 'main',
      title: 'Main',
      options: [
        {
          index: 0,
          kind: 'bool',
          fieldId: 'On',
          name: 'On',
          tooltip: '',
          value: false,
          min: null,
          max: null,
          interval: null,
          choices: null,
          formatSamples: null,
          editable: true,
          titleScreenOnly: false,
        },
        {
          index: 1,
          kind: 'int',
          fieldId: 'N',
          name: 'N',
          tooltip: '',
          value: 2,
          min: 0,
          max: 10,
          interval: 1,
          choices: null,
          formatSamples: ['2'],
          editable: true,
          titleScreenOnly: false,
        },
      ],
    },
  ],
}

describe('menuControl', () => {
  test('maps kinds', () => {
    expect(menuControl('bool')).toBe('switch')
    expect(menuControl('int')).toBe('number')
    expect(menuControl('float')).toBe('number')
    expect(menuControl('text')).toBe('text')
    expect(menuControl('choice')).toBe('select')
    expect(menuControl('keybind')).toBe('keybind')
    expect(menuControl('keybindList')).toBe('keybind')
    expect(menuControl('color')).toBe('color')
    expect(menuControl('image')).toBe('image')
    expect(menuControl('sectionTitle')).toBe('section')
    expect(menuControl('subHeader')).toBe('subHeader')
    expect(menuControl('paragraph')).toBe('paragraph')
    expect(menuControl('pageLink')).toBe('pageLink')
    expect(menuControl('complex')).toBe('readonly')
    expect(menuControl('image display')).toBe('readonly')
    expect(editableKind('complex')).toBe(false)
    expect(editableKind('bool')).toBe(true)
  })
})

describe('pendingEdits', () => {
  test('emits only changed options', () => {
    expect(pendingEdits(capture, { 'main/0': false })).toEqual([])
    expect(pendingEdits(capture, { 'main/0': true })).toEqual([
      { page: 'main', index: 0, kind: 'bool', fieldId: 'On', name: 'On', value: true },
    ])
    expect(valuesEqual(2, 2)).toBe(true)
    expect(
      draftMap({
        schema: 1,
        edits: [{ page: 'main', index: 1, kind: 'int', fieldId: 'N', name: 'N', value: 9 }],
      }),
    ).toEqual({
      'main/1': 9,
    })
  })
})
