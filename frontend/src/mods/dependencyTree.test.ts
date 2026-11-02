import { describe, expect, test } from 'bun:test'
import {
  annotateTree,
  buildDependencyTrees,
  childrenOf,
  manifestsFromMods,
  type ProfileManifest,
  type TreeNode,
} from './dependencyTree.ts'

const pack: ProfileManifest = {
  id: 'Pathoschild.ContentPatcher',
  dependencies: [{ id: 'SMAPI', isRequired: true }],
}

const shape = (
  nodes: readonly { id: string; required: boolean; cycle: boolean; count: number }[],
) => nodes.map((n) => [n.id, n.required, n.cycle, n.count])

describe('buildDependencyTrees', () => {
  test('ContentPackFor is a required edge onto the framework, expanded lazily', () => {
    const manifests = [pack, { id: 'Example.Pack', contentPackFor: 'Pathoschild.ContentPatcher' }]
    const trees = buildDependencyTrees(manifests, 'Example.Pack')
    expect(shape(trees.needs)).toEqual([['Pathoschild.ContentPatcher', true, false, 1]])
    expect(trees.neededBy).toEqual([])
    const [cp] = trees.needs
    expect(shape(childrenOf(manifests, 'needs', cp as TreeNode))).toEqual([
      ['SMAPI', true, false, 0],
    ])
  })

  test('optional vs required edges stay marked; missing nodes have no children', () => {
    const trees = buildDependencyTrees(
      [
        {
          id: 'A.Mod',
          dependencies: [{ id: 'B.Required' }, { id: 'C.Optional', isRequired: false }],
        },
      ],
      'A.Mod',
    )
    expect(shape(trees.needs)).toEqual([
      ['B.Required', true, false, 0],
      ['C.Optional', false, false, 0],
    ])
  })

  test('a cycle is flagged on the node that loops back and has no children', () => {
    const manifests = [
      { id: 'A.Mod', dependencies: [{ id: 'B.Mod' }] },
      { id: 'B.Mod', dependencies: [{ id: 'A.Mod' }] },
    ]
    const trees = buildDependencyTrees(manifests, 'A.Mod')
    expect(shape(trees.needs)).toEqual([['B.Mod', true, false, 1]])
    const below = childrenOf(manifests, 'needs', trees.needs[0] as TreeNode)
    expect(shape(below)).toEqual([['A.Mod', true, true, 0]])
    expect(childrenOf(manifests, 'needs', below[0] as TreeNode)).toEqual([])
    const up = buildDependencyTrees(manifests, 'A.Mod').neededBy
    expect(shape(childrenOf(manifests, 'neededBy', up[0] as TreeNode))).toEqual([
      ['A.Mod', true, true, 0],
    ])
  })

  test('each dependent is listed once per level however many paths reach it', () => {
    const manifests = [
      { id: 'Core' },
      { id: 'Mid.One', dependencies: [{ id: 'Core' }] },
      { id: 'Mid.Two', dependencies: [{ id: 'Core' }, { id: 'Mid.One' }] },
      { id: 'Leaf', dependencies: [{ id: 'Mid.One' }, { id: 'Mid.Two' }, { id: 'mid.one' }] },
    ]
    const { neededBy } = buildDependencyTrees(manifests, 'Core')
    expect(shape(neededBy)).toEqual([
      ['Mid.One', true, false, 2],
      ['Mid.Two', true, false, 1],
    ])
    const two = neededBy[1] as TreeNode
    expect(shape(childrenOf(manifests, 'neededBy', two))).toEqual([['Leaf', true, false, 0]])
    const leaf = buildDependencyTrees(manifests, 'Leaf').needs
    expect(shape(leaf)).toEqual([
      ['Mid.One', true, false, 1],
      ['Mid.Two', true, false, 2],
    ])
  })

  test('needed-by walks incoming edges across the profile', () => {
    const trees = buildDependencyTrees(
      [
        pack,
        { id: 'Farm.Pack', contentPackFor: 'Pathoschild.ContentPatcher' },
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
