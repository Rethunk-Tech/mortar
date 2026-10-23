import { expect, test } from 'bun:test'
import {
  type SMAPIProblem,
  SMAPIProblemKind,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { runProblemToken } from './runProblemToken.ts'

test('a run problem token matches the one internal/problems stores', () => {
  const problem = {
    kind: SMAPIProblemKind.SMAPIProblemDuplicate,
    modId: 'HatLib',
    modName: 'HatLib',
    detail: 'a newer copy loaded instead: HatLib 0.0.9',
  } as SMAPIProblem
  expect(runProblemToken(problem)).toBe('runProblem\tduplicate\tHatLib')
})
