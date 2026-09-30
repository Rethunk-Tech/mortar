import { describe, expect, test } from 'bun:test'
import {
  annotateTree,
  buildDependencyTrees,
  manifestsFromMods,
  type ProfileManifest,
} from './dependencyTree.ts'

const pack: ProfileManifest = {
  uniqueID: 'Pathoschild.ContentPatcher',
  dependencies: [{ uniqueID: 'SMAPI', isRequired: true }],
}

describe('buildDependencyTrees', () => {
  test('ContentPackFor is a required edge onto the framework', () => {
    const trees = buildDependencyTrees(
      [
        pack,
        {
          uniqueID: 'Example.Pack',
          contentPackFor: 'Pathoschild.ContentPatcher',
        },
      ],
      'Example.Pack',
    )
    expect(trees.needs).toEqual([
      {
        uniqueID: 'Pathoschild.ContentPatcher',
        required: true,
        cycle: false,
        children: [
          {
            uniqueID: 'SMAPI',
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
          uniqueID: 'A.Mod',
          dependencies: [{ uniqueID: 'B.Required' }, { uniqueID: 'C.Optional', isRequired: false }],
        },
      ],
      'A.Mod',
    )
    expect(trees.needs).toEqual([
      {
        uniqueID: 'B.Required',
        required: true,
        cycle: false,
        children: [],
      },
      {
        uniqueID: 'C.Optional',
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
          uniqueID: 'A.Mod',
          dependencies: [{ uniqueID: 'B.Mod' }],
        },
        {
          uniqueID: 'B.Mod',
          dependencies: [{ uniqueID: 'A.Mod' }],
        },
      ],
      'A.Mod',
    )
    expect(trees.needs).toEqual([
      {
        uniqueID: 'B.Mod',
        required: true,
        cycle: false,
        children: [
          {
            uniqueID: 'A.Mod',
            required: true,
            cycle: true,
            children: [],
          },
        ],
      },
    ])
    expect(trees.neededBy).toEqual([
      {
        uniqueID: 'B.Mod',
        required: true,
        cycle: false,
        children: [
          {
            uniqueID: 'A.Mod',
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
          uniqueID: 'Farm.Pack',
          contentPackFor: 'Pathoschild.ContentPatcher',
        },
        {
          uniqueID: 'Town.Pack',
          contentPackFor: 'Pathoschild.ContentPatcher',
          dependencies: [{ uniqueID: 'Extra.Lib', isRequired: false }],
        },
      ],
      'Pathoschild.ContentPatcher',
    )
    expect(trees.neededBy.map((n) => n.uniqueID).toSorted((a, b) => a.localeCompare(b))).toEqual([
      'Farm.Pack',
      'Town.Pack',
    ])
    expect(trees.neededBy.find((n) => n.uniqueID === 'Town.Pack')).toEqual({
      uniqueID: 'Town.Pack',
      required: true,
      cycle: false,
      children: [],
    })
  })

  test('UniqueIDs match case-insensitively and keep the first spelling', () => {
    const trees = buildDependencyTrees(
      [
        {
          uniqueID: 'SpaceCode',
          dependencies: [{ uniqueID: 'json.assets', isRequired: false }],
        },
        {
          uniqueID: 'Json.Assets',
        },
      ],
      'spacecode',
    )
    expect(trees.needs[0]?.uniqueID).toBe('Json.Assets')
    expect(trees.neededBy).toEqual([])
  })
})

describe('manifestsFromMods', () => {
  test('marks UniqueIDs in optional as not required', () => {
    expect(
      manifestsFromMods([
        {
          uniqueId: 'A.Mod',
          needs: ['B.Req', 'C.Opt'],
          optional: ['C.Opt'],
        },
      ]),
    ).toEqual([
      {
        uniqueID: 'A.Mod',
        dependencies: [
          { uniqueID: 'B.Req', isRequired: true },
          { uniqueID: 'C.Opt', isRequired: false },
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
          uniqueID: 'Root',
          dependencies: [
            { uniqueID: 'On.Mod' },
            { uniqueID: 'Off.Mod' },
            { uniqueID: 'Gone.Mod' },
            { uniqueID: 'Broke.Mod' },
          ],
        },
      ],
      'Root',
    )
    const view = annotateTree(trees.needs, [
      { uniqueID: 'On.Mod', enabled: true },
      { uniqueID: 'Off.Mod', enabled: false },
      { uniqueID: 'Broke.Mod', enabled: true, broken: true },
    ])
    expect(view.map((n) => [n.uniqueID, n.state])).toEqual([
      ['On.Mod', 'enabled'],
      ['Off.Mod', 'disabled'],
      ['Gone.Mod', 'missing'],
      ['Broke.Mod', 'broken'],
    ])
  })
})
