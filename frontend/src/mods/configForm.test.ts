import { expect, test } from 'bun:test'
import { type ConfigNode, parseConfig, setAt, stringifyConfig } from './configForm.ts'

test('parseConfig keeps key order, ints, floats, nested objects and string lists', () => {
  const node = parseConfig(
    `{"z":1,"n":1.5,"on":true,"s":"hi","xs":["a","b"],"nested":{"k":2},"mixed":[1,"x"],"empty":null}`,
  )
  expect(node).toEqual({
    kind: 'object',
    entries: [
      { key: 'z', node: { kind: 'int', value: 1 } },
      { key: 'n', node: { kind: 'float', value: 1.5 } },
      { key: 'on', node: { kind: 'bool', value: true } },
      { key: 's', node: { kind: 'string', value: 'hi' } },
      { key: 'xs', node: { kind: 'strings', value: ['a', 'b'] } },
      {
        key: 'nested',
        node: { kind: 'object', entries: [{ key: 'k', node: { kind: 'int', value: 2 } }] },
      },
      { key: 'mixed', node: { kind: 'readonly', json: '[1,"x"]' } },
      { key: 'empty', node: { kind: 'readonly', json: 'null' } },
    ],
  })
  expect(stringifyConfig(node)).toBe(`{
  "z": 1,
  "n": 1.5,
  "on": true,
  "s": "hi",
  "xs": [
    "a",
    "b"
  ],
  "nested": {
    "k": 2
  },
  "mixed": [
    1,
    "x"
  ],
  "empty": null
}
`)
})

test('setAt replaces a nested field without shuffling siblings', () => {
  const root = parseConfig(`{"a":{"x":1,"y":2},"b":true}`)
  const next = setAt(root, [0, 1], { kind: 'int', value: 9 })
  const obj = next as Extract<ConfigNode, { kind: 'object' }>
  expect(obj.entries.map((e) => e.key)).toEqual(['a', 'b'])
  const nested = obj.entries[0]?.node as Extract<ConfigNode, { kind: 'object' }>
  expect(nested.entries.map((e) => e.key)).toEqual(['x', 'y'])
  expect(nested.entries[1]?.node).toEqual({ kind: 'int', value: 9 })
})
