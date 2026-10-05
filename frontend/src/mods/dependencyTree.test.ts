import { describe, expect, test } from 'bun:test'
import {
  annotateTree,
  buildDependencyTrees,
  manifestsFromMods,
  type ProfileManifest,
} from './dependencyTree.ts'

const pack: ProfileManifest = {
  id: 'Pathoschild.ContentPatcher',
  dependencies: [{ id: 'SMAPI', isRequired: true }],
}

describe('buildDependencyTrees', () => {
  test('ContentPackFor is a required edge onto the framework', () => {
    const trees = buildDependencyTrees(
      [
        pack,
        {
          id: 'Example.Pack',
          contentPackFor: 'Pathoschild.ContentPatcher',
        },
      ],
      'Example.Pack',
    )
    expect(trees.needs).toEqual([
      {
        id: 'Pathoschild.ContentPatcher',
        required: true,
        cycle: false,
        children: [
          {
            id: 'SMAPI',
            required: true,
            cycle: false,
            children: [],
          },
        ],
      },
    ])
    expect(trees.neededBy).toEqual([])
  })

  test('optional vs required edges stay marked; missing nodes appear with no children', () => {
    const trees = buildDependencyTrees(
      [
        {
          id: 'A.Mod',
          dependencies: [{ id: 'B.Required' }, { id: 'C.Optional', isRequired: false }],
        },
      ],
      'A.Mod',
    )
    expect(trees.needs).toEqual([
      {
        id: 'B.Required',
        required: true,
        cycle: false,
        children: [],
      },
      {
        id: 'C.Optional',
        required: false,
        cycle: false,
        children: [],
      },
    ])
  })

  test('cycles are cut and the looping id is shown once', () => {
    const trees = buildDependencyTrees(
      [
        {
          id: 'A.Mod',
          dependencies: [{ id: 'B.Mod' }],
        },
        {
          id: 'B.Mod',
          dependencies: [{ id: 'A.Mod' }],
        },
      ],
      'A.Mod',
    )
    expect(trees.needs).toEqual([
      {
        id: 'B.Mod',
        required: true,
        cycle: false,
        children: [
          {
            id: 'A.Mod',
            required: true,
            cycle: true,
            children: [],
          },
        ],
      },
    ])
    expect(trees.neededBy).toEqual([
      {
        id: 'B.Mod',
        required: true,
        cycle: false,
        children: [
          {
            id: 'A.Mod',
            required: true,
            cycle: true,
            children: [],
          },
        ],
      },
    ])
  })

  test('needed-by walks incoming edges across the profile', () => {
    const trees = buildDependencyTrees(
      [
        pack,
        {
          id: 'Farm.Pack',
          contentPackFor: 'Pathoschild.ContentPatcher',
        },
        {
          id: 'Town.Pack',
          contentPackFor: 'Pathoschild.ContentPatcher',
          dependencies: [{ id: 'Extra.Lib', isRequired: false }],
        },
      ],
      'Pathoschild.ContentPatcher',
    )
    expect(trees.neededBy.map((n) => n.id).toSorted((a, b) => a.localeCompare(b))).toEqual([
      'Farm.Pack',
      'Town.Pack',
    ])
    expect(trees.neededBy.find((n) => n.id === 'Town.Pack')).toEqual({
      id: 'Town.Pack',
      required: true,
      cycle: false,
      children: [],
    })
  })

  test('UniqueIDs match case-insensitively and keep the first spelling', () => {
    const trees = buildDependencyTrees(
      [
        {
          id: 'SpaceCode',
          dependencies: [{ id: 'json.assets', isRequired: false }],
        },
        {
          id: 'Json.Assets',
        },
      ],
      'spacecode',
    )
    expect(trees.needs[0]?.id).toBe('Json.Assets')
    expect(trees.neededBy).toEqual([])
  })
})

describe('manifestsFromMods', () => {
  test('marks UniqueIDs in optional as not required', () => {
    expect(
      manifestsFromMods([
        {
          id: 'A.Mod',
          needs: ['B.Req', 'C.Opt'],
          optional: ['C.Opt'],
        },
      ]),
    ).toEqual([
      {
        id: 'A.Mod',
        dependencies: [
          { id: 'B.Req', isRequired: true },
          { id: 'C.Opt', isRequired: false },
        ],
      },
    ])
  })
})

describe('annotateTree', () => {
  test('marks enabled, disabled, missing, and broken', () => {
    const trees = buildDependencyTrees(
      [
        {
          id: 'Root',
          dependencies: [
            { id: 'On.Mod' },
            { id: 'Off.Mod' },
            { id: 'Gone.Mod' },
            { id: 'Broke.Mod' },
          ],
        },
      ],
      'Root',
    )
    const view = annotateTree(trees.needs, [
      { id: 'On.Mod', enabled: true },
      { id: 'Off.Mod', enabled: false },
      { id: 'Broke.Mod', enabled: true, broken: true },
    ])
    expect(view.map((n) => [n.id, n.state])).toEqual([
      ['On.Mod', 'enabled'],
      ['Off.Mod', 'disabled'],
      ['Gone.Mod', 'missing'],
      ['Broke.Mod', 'broken'],
    ])
  })
})
