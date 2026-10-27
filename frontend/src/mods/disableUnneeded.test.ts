import { expect, test } from 'bun:test'
import type {
  Entry,
  Mod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { dependentsStillOn, disableAlso, unneededImplicit } from './disableUnneeded.ts'

const mod = (id: string, needs: string[] = [], over: Partial<Mod> = {}) =>
  ({ key: id, id, name: id, enabled: true, needs, ...over }) as Mod

test('switching a mod off names the implicit mods only it needed, through what they need', () => {
  const pack = mod('Me.Pack', ['Lib.Core', 'Lib.Shared', 'Me.Chosen', 'Lib.Extra'])
  const all = [
    pack,
    mod('Lib.Extra'),
    mod('Uses.Extra', ['Lib.Extra'], { optional: ['Lib.Extra'] }),
    mod('Lib.Core', ['Lib.Deep']),
    mod('Lib.Deep'),
    mod('Lib.Shared'),
    mod('Other.Mod', ['Lib.Shared']),
    mod('Me.Chosen'),
  ]
  const implicit = (m: Mod) => m.id.startsWith('Lib.')
  expect(unneededImplicit(all, [pack], implicit).map((m) => m.id)).toEqual(['Lib.Core', 'Lib.Deep'])
})

test('removing also takes switched-off libraries, and keeps what a switched-off mod that stays needs', () => {
  const pack = mod('Me.Pack', ['Lib.Off', 'Lib.Kept'])
  const all = [
    pack,
    mod('Lib.Off', [], { enabled: false }),
    mod('Lib.Kept'),
    mod('Me.Resting', ['Lib.Kept'], { enabled: false }),
  ]
  const implicit = (m: Mod) => m.id.startsWith('Lib.')
  expect(unneededImplicit(all, [pack], implicit, true).map((m) => m.id)).toEqual(['Lib.Off'])
  // Switching off instead leaves a library only switched-off mods need free to go off too.
  expect(unneededImplicit(all, [pack], implicit).map((m) => m.id)).toEqual(['Lib.Kept'])
})

test('switching a mod off finds the enabled mods that need it, through what needs them in turn', () => {
  const core = mod('Core.Lib', [], { enabled: false })
  const all = [
    core,
    mod('Uses.Core', ['Core.Lib']),
    mod('Uses.Uses', ['Uses.Core']),
    mod('Optional.User', ['Core.Lib'], { optional: ['Core.Lib'] }),
    mod('Already.Off', ['Core.Lib'], { enabled: false }),
    mod('Unrelated'),
  ]
  expect(dependentsStillOn(all, [core]).map((m) => m.id)).toEqual(['Uses.Core', 'Uses.Uses'])
})

test('disabling asks first, takes the dependents and lone libraries along, or leaves them, per the preference', () => {
  const before = useProfiles.getState()
  const all = [mod('Me.Ui', ['Me.Detour']), mod('Me.Detour', ['Lib.Core']), mod('Lib.Core')]
  const off = [all[1] as Mod]
  const open = (mode: string) =>
    useProfiles.setState({
      openId: 'p1',
      profiles: [
        {
          ...({} as (typeof before.profiles)[number]),
          id: 'p1',
          entries: [
            {
              key: 'Lib.Core',
              implicit: true,
            } as Entry,
          ],
          overrides: { disableDependencies: mode },
        },
      ],
    })
  try {
    open('ask')
    expect(disableAlso(all, off)).toBeNull()
    expect(disableAlso(all, [all[0] as Mod])).toEqual([])
    open('always')
    expect(disableAlso(all, off)?.map((m) => m.id)).toEqual(['Me.Ui', 'Lib.Core'])
    open('never')
    expect(disableAlso(all, off)).toEqual([])
  } finally {
    useProfiles.setState(before)
  }
})
