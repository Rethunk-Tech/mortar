import { expect, test } from 'bun:test'
import {
  SMAPIFixKind,
  type SMAPIProblem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'
import { stillApplies } from './runProblemsLive.ts'

const problem = (over: Partial<SMAPIProblem>): SMAPIProblem =>
  ({
    kind: 'incompatible',
    modId: '',
    modName: '',
    detail: '',
    fix: SMAPIFixKind.SMAPIFixUpdate,
    ...over,
  }) as SMAPIProblem

const mods = [
  { uniqueId: 'spacechase0.CookingSkill', name: 'Cooking Skill', version: '1.5.0', enabled: true },
  { uniqueId: 'a.Old', name: 'Old Thing', version: '1.0.0', enabled: false },
]

test('a run problem drops once the mod is removed, updated, or already switched off', () => {
  expect(stillApplies(problem({ modName: 'Non Destructive NPCs 1.0.0' }), mods)).toBe(false)
  expect(stillApplies(problem({ modName: 'Cooking Skill 1.4.5' }), mods)).toBe(false)
  expect(stillApplies(problem({ modName: 'Cooking Skill 1.5.0' }), mods)).toBe(true)
  expect(
    stillApplies(problem({ modName: 'Old Thing 1.0.0', fix: SMAPIFixKind.SMAPIFixDisable }), mods),
  ).toBe(false)
  expect(
    stillApplies(
      problem({
        fix: SMAPIFixKind.SMAPIFixInstallDependency,
        dependency: 'spacechase0.CookingSkill',
      }),
      mods,
    ),
  ).toBe(false)
  expect(
    stillApplies(
      problem({ fix: SMAPIFixKind.SMAPIFixInstallDependency, dependency: 'x.Missing' }),
      mods,
    ),
  ).toBe(true)
})
