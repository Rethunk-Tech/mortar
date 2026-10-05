import { idKey } from './dependents.ts'

interface Edge {
  to: string
  required: boolean
}

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

function walk(from: string, adj: Map<string, Edge[]>, stack: Set<string>): TreeNode[] {
  const children: TreeNode[] = []
  for (const e of adj.get(from) ?? []) {
    const key = idKey(e.to)
    const cycle = stack.has(key)
    if (cycle) {
      children.push({ id: e.to, required: e.required, cycle: true, children: [] })
    } else {
      stack.add(key)
      children.push({
        id: e.to,
        required: e.required,
        cycle: false,
        children: walk(e.to, adj, stack),
      })
      stack.delete(key)
    }
  }
  return children
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

export interface ProfileManifest {
  id: string
  dependencies?: readonly ManifestDep[]
  contentPackFor?: string
}

export interface TreeNode {
  id: string
  required: boolean
  cycle: boolean
  children: TreeNode[]
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
  state: DepNodeState
  children: DepViewNode[]
}

export function buildDependencyTrees(
  manifests: readonly ProfileManifest[],
  rootID: string,
): DependencyTrees {
  const { edges, spellings } = outgoing(manifests)
  const root = canonical(spellings, rootID)
  const stack = new Set<string>([idKey(root)])
  return {
    needs: walk(root, edges, stack),
    neededBy: walk(root, invert(edges), stack),
  }
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
  return nodes.map((n) => ({
    id: n.id,
    required: n.required,
    cycle: n.cycle,
    state: nodeState(n.id, byID),
    children: annotateTree(n.children, installed),
  }))
}
