import { afterEach, expect, test } from 'bun:test'
import { useToasts } from '../toasts/store.ts'
import { showRescued } from './rescuedToast.ts'

afterEach(() => {
  useToasts.setState(useToasts.getInitialState(), true)
})

test('the rescued event toasts restored edits with a discard action', () => {
  useToasts.setState({ toasts: [] })
  showRescued({ paths: [] })
  expect(useToasts.getState().toasts).toHaveLength(0)
  showRescued({ paths: ['/game/a.cfg.mortar-rescued'] })
  const [toast] = useToasts.getState().toasts
  expect(toast?.title).toBe('Edits were restored')
  expect(toast?.action?.label).toBe('Discard')
})
