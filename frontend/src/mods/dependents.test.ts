import { expect, test } from 'bun:test'
import { dependentsOf, requiredIdsOf, requirementKeys } from './dependents.ts'

test('requiredIdsOf drops optional needs and keeps ContentPackFor', () => {
  expect(
    requiredIdsOf({
      needs: ['Core.Lib', 'Nice.ToHave'],
      optional: ['Nice.ToHave'],
      contentPackFor: 'Pathoschild.ContentPatcher',
    }),
  ).toEqual(['Core.Lib', 'Pathoschild.ContentPatcher'])
})

test('dependentsOf lists enabled mods that require a removed UniqueID', () => {
  const core = { id: 'Core.Lib', name: 'Core', enabled: true, needs: [] as string[] }
  const pack = {
    id: 'Farm.Pack',
    name: 'Farm pack',
    enabled: true,
    needs: ['Core.Lib'],
    optional: [] as string[],
  }
  const off = {
    id: 'Off.Pack',
    name: 'Off',
    enabled: false,
    needs: ['Core.Lib'],
  }
  const extra = {
    id: 'Extra.Opt',
    name: 'Optional user',
    enabled: true,
    needs: ['Core.Lib'],
    optional: ['Core.Lib'],
  }
  expect(dependentsOf([core, pack, off, extra], [core]).map((m) => m.id)).toEqual(['Farm.Pack'])
})

test('a group made from mods can bring along everything they need, through every level', () => {
  const mods = [
    { id: 'a', key: 'ka', needs: ['b'] },
    { id: 'b', key: 'kb', needs: ['c', 'opt'], optional: ['opt'] },
    { id: 'c', key: 'kc' },
    { id: 'opt', key: 'ko' },
    { id: 'd', key: 'kd', needs: ['a'] },
  ]
  expect(
    requirementKeys(
      mods,
      [mods[0], mods[4]].filter((m) => m !== undefined),
    ),
  ).toEqual(['kb', 'kc'])
})
