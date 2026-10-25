import { expect, test } from 'bun:test'
import { tokenize } from './codeTokens.ts'

const kinds = (line: string, il: boolean) =>
  tokenize(line, il)
    .filter((t) => t.text.trim() !== '')
    .map((t) => `${t.kind}:${t.text.trim()}`)

test('C# keywords, types, strings, numbers and comments', () => {
  expect(kinds('public static int Count = 42; // the "count"', false)).toEqual([
    'keyword:public static int',
    'type:Count',
    'plain:=',
    'number:42',
    'plain:;',
    'comment:// the "count"',
  ])
  expect(kinds('Log("a \\" b");', false)).toEqual([
    'type:Log',
    'plain:(',
    'string:"a \\" b"',
    'plain:);',
  ])
})

test('IL labels, opcodes and directives', () => {
  expect(kinds('IL_0008: stfld class Item Items::item', true)).toEqual([
    'label:IL_0008:',
    'keyword:stfld class',
    'plain:Item Items::item',
  ])
  expect(kinds('.method public hidebysig static', true)).toEqual([
    'directive:.method',
    'keyword:public hidebysig static',
  ])
})
