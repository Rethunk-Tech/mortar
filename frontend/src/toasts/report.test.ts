import { beforeEach, expect, test } from 'bun:test'
import { reportUnexpected, toastError } from './report.ts'
import { useToasts } from './store.ts'

const last = () => useToasts.getState().toasts.at(-1)

beforeEach(() => useToasts.setState({ toasts: [], history: [] }))

test('an unexpected failure is titled by its cause, not a generic sentence', () => {
  reportUnexpected(new Error("GitHub's release API answered 404 Not Found"))
  expect(last()?.title).toBe("GitHub's release API answered 404 Not Found")
  expect(last()?.body).toBeUndefined()
  expect(last()?.detail).toBeUndefined()
})

test('an untagged failure under a named title carries its cause as the detail, with no generic body', () => {
  toastError('Could not save that setting', new Error('open settings.json: read-only file system'))
  expect(last()?.body).toBeUndefined()
  expect(last()?.detail).toBe('open settings.json: read-only file system')
})

test('a game the system will not let Mortar end gets its own wording, not the storage settings', () => {
  toastError('Could not stop the game', new Error('[process_denied] Access is denied.'))
  expect(last()?.action?.label).toBe('Open log')
  toastError('Could not save', new Error('[permission] open settings.json'))
  expect(last()?.action?.label).toBe('Open storage settings')
})
