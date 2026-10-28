import { afterAll, expect, test } from 'bun:test'

const testing = await import('@testing-library/react')
const { readStored, useStoredState } = await import('./useStoredState.ts')

afterAll(() => {
  testing.cleanup()
})

const isText = (v: unknown): v is string => typeof v === 'string'

test('a stored string reads back whether it was written as JSON or bare', () => {
  localStorage.setItem('stored-test.json', JSON.stringify('run-1'))
  localStorage.setItem('stored-test.bare', 'run-2')
  expect(readStored('stored-test.json', '', isText)).toBe('run-1')
  expect(readStored('stored-test.bare', '', isText)).toBe('run-2')
  expect(readStored('stored-test.missing', 'none', isText)).toBe('none')
  localStorage.removeItem('stored-test.json')
  localStorage.removeItem('stored-test.bare')
})

test('the value follows the key when the key changes', () => {
  localStorage.setItem('stored-test.a', JSON.stringify('one'))
  localStorage.setItem('stored-test.b', JSON.stringify('two'))
  const view = testing.renderHook(({ key }) => useStoredState(key, '', isText), {
    initialProps: { key: 'stored-test.a' },
  })
  expect(view.result.current[0]).toBe('one')
  view.rerender({ key: 'stored-test.b' })
  expect(view.result.current[0]).toBe('two')
  testing.act(() => view.result.current[1]('three'))
  expect(localStorage.getItem('stored-test.b')).toBe('"three"')
  expect(localStorage.getItem('stored-test.a')).toBe('"one"')
  localStorage.removeItem('stored-test.a')
  localStorage.removeItem('stored-test.b')
})
