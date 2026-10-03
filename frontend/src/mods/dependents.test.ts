import { expect, test } from 'bun:test'
import { dependentsOf, requiredIdsOf } from './dependents.ts'

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
  const core = { uniqueId: 'Core.Lib', name: 'Core', enabled: true, needs: [] as string[] }
  const pack = {
    uniqueId: 'Farm.Pack',
    name: 'Farm pack',
    enabled: true,
    needs: ['Core.Lib'],
    optional: [] as string[],
  }
  const off = {
    uniqueId: 'Off.Pack',
    name: 'Off',
    enabled: false,
    needs: ['Core.Lib'],
  }
  const extra = {
    uniqueId: 'Extra.Opt',
    name: 'Optional user',
    enabled: true,
    needs: ['Core.Lib'],
    optional: ['Core.Lib'],
  }
  expect(dependentsOf([core, pack, off, extra], [core]).map((m) => m.uniqueId)).toEqual([
    'Farm.Pack',
  ])
})
