import { idKey } from './dependents.ts'

interface Edge {
  to: string
  required: boolean
}

// One graph per manifest list, so walking it from another root does not rebuild it.
const graphs = new WeakMap<
  readonly ProfileManifest[],
  { edges: Map<string, Edge[]>; back: Map<string, Edge[]>; spellings: Map<string, string> }
>()

function canonical(spellings: Map<string, string>, raw: string): string {
  const key = idKey(raw)
  const existing = spellings.get(key)
  if (existing !== undefined) {
    return existing
  }
  const trimmed = raw.trim()
  spellings.set(key, trimmed)
  return trimmed
}

function outgoing(manifests: readonly ProfileManifest[]): {
  edges: Map<string, Edge[]>
  spellings: Map<string, string>
} {
  const spellings = new Map<string, string>()
  const edges = new Map<string, Edge[]>()

  for (const man of manifests) {
    canonical(spellings, man.id)
  }
  for (const man of manifests) {
    for (const dep of man.dependencies ?? []) {
      canonical(spellings, dep.id)
    }
    if (man.contentPackFor) {
      canonical(spellings, man.contentPackFor)
    }
  }

  const add = (fromRaw: string, toRaw: string, required: boolean) => {
    const from = canonical(spellings, fromRaw)
    const to = canonical(spellings, toRaw)
    if (idKey(from) === idKey(to)) {
      return
    }
    const list = edges.get(from) ?? []
    const dup = list.find((e) => idKey(e.to) === idKey(to))
    if (dup) {
      dup.required = dup.required || required
      return
    }
    list.push({ to, required })
    edges.set(from, list)
  }

  for (const man of manifests) {
    for (const dep of man.dependencies ?? []) {
      add(man.id, dep.id, dep.isRequired !== false)
    }
    if (man.contentPackFor) {
      add(man.id, man.contentPackFor, true)
    }
  }

  return { edges, spellings }
}

function invert(edges: Map<string, Edge[]>): Map<string, Edge[]> {
  const back = new Map<string, Edge[]>()
  for (const [from, list] of edges) {
    for (const e of list) {
      const incoming = back.get(e.to) ?? []
      const dup = incoming.find((x) => idKey(x.to) === idKey(from))
      if (dup) {
        dup.required = dup.required || e.required
      } else {
        incoming.push({ to: from, required: e.required })
        back.set(e.to, incoming)
      }
    }
  }
  return back
}

type Side = 'needs' | 'neededBy'

function graphOf(manifests: readonly ProfileManifest[]) {
  let graph = graphs.get(manifests)
  if (!graph) {
    const { edges, spellings } = outgoing(manifests)
    graph = { edges, back: invert(edges), spellings }
    graphs.set(manifests, graph)
  }
  return graph
}

// One level only: a node's own children are built when it is expanded, since the full expansion of a framework's
// dependents repeats each dependent once per path through the graph.
function level(parent: string, path: readonly string[], adj: Map<string, Edge[]>): TreeNode[] {
  return (adj.get(parent) ?? []).map((e) => {
    const key = idKey(e.to)
    const cycle = path.includes(key)
    return {
      id: e.to,
      required: e.required,
      cycle,
      count: cycle ? 0 : (adj.get(e.to)?.length ?? 0),
      path: [...path, key],
    }
  })
}

function nodeState(id: string, byID: Map<string, InstalledModState>): DepNodeState {
  const inst = byID.get(idKey(id))
  if (!inst) {
    return 'missing'
  }
  if (inst.broken) {
    return 'broken'
  }
  return inst.enabled ? 'enabled' : 'disabled'
}

interface ManifestDep {
  id: string
  isRequired?: boolean
}

type DepNodeState = 'enabled' | 'disabled' | 'missing' | 'broken'

function annotate(nodes: readonly TreeNode[], byID: Map<string, InstalledModState>): DepViewNode[] {
  return nodes.map((n) => ({
    id: n.id,
    required: n.required,
    cycle: n.cycle,
    count: n.count,
    path: n.path,
    state: nodeState(n.id, byID),
  }))
}

export interface ProfileManifest {
  id: string
  dependencies?: readonly ManifestDep[]
  contentPackFor?: string
}

export interface TreeNode {
  id: string
  required: boolean
  cycle: boolean
  /** Edges out of this node on its side of the graph: what expanding it lists. Zero for a cycle. */
  count: number
  /** Id keys from the root down to this node, which is what marks a child as a cycle. */
  path: readonly string[]
}

export interface DependencyTrees {
  needs: TreeNode[]
  neededBy: TreeNode[]
}

export interface InstalledModState {
  id: string
  enabled: boolean
  broken?: boolean
}

export interface DepViewNode {
  id: string
  required: boolean
  cycle: boolean
  count: number
  path: readonly string[]
  state: DepNodeState
}

export function buildDependencyTrees(
  manifests: readonly ProfileManifest[],
  rootID: string,
): DependencyTrees {
  const graph = graphOf(manifests)
  const root = canonical(graph.spellings, rootID)
  const path = [idKey(root)]
  return {
    needs: level(root, path, graph.edges),
    neededBy: level(root, path, graph.back),
  }
}

export function childrenOf(
  manifests: readonly ProfileManifest[],
  side: Side,
  node: TreeNode,
): TreeNode[] {
  if (node.cycle) {
    return []
  }
  const graph = graphOf(manifests)
  return level(node.id, node.path, side === 'needs' ? graph.edges : graph.back)
}

export function manifestsFromMods(
  mods: readonly {
    id: string
    needs?: string[] | null
    optional?: string[] | null
  }[],
): ProfileManifest[] {
  return mods.map((m) => {
    const optional = new Set((m.optional ?? []).map(idKey))
    return {
      id: m.id,
      dependencies: (m.needs ?? []).map((id) => ({
        id,
        isRequired: !optional.has(idKey(id)),
      })),
    }
  })
}

export function annotateTree(
  nodes: readonly TreeNode[],
  installed: readonly InstalledModState[],
): DepViewNode[] {
  const byID = new Map<string, InstalledModState>()
  for (const m of installed) {
    byID.set(idKey(m.id), m)
  }
  return annotate(nodes, byID)
}
