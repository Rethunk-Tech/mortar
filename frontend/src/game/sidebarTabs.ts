import type { TabId } from './tab.ts'

type SidebarGroupId = 'library' | 'get' | 'health' | 'logs'

// What the open profile's loader supports; a section the loader lacks has no entry.
interface SidebarCaps {
  order: boolean
  console: boolean
  startup: boolean
}

interface SidebarCounts {
  updates: number
  // Null until the problems are first read.
  problems: number | null
}

interface SidebarEntry {
  id: TabId
  badge: { n: number; tone: 'primary' | 'warning' } | null
}

interface SidebarGroup {
  id: SidebarGroupId
  entries: SidebarEntry[]
}

const count = (n: number | null, tone: 'primary' | 'warning'): SidebarEntry['badge'] =>
  n !== null && n > 0 ? { n, tone } : null

// The sidebar's groups under Home, in order: Mods carries the update count and Problems the problem count; a group
// whose sections the loader lacks is left out, header and all.
function sidebarGroups(caps: SidebarCaps, counts: SidebarCounts): SidebarGroup[] {
  const groups: SidebarGroup[] = [
    {
      id: 'library',
      entries: [
        { id: 'mods', badge: count(counts.updates, 'primary') },
        { id: 'saves', badge: null },
      ],
    },
    { id: 'get', entries: [{ id: 'browse', badge: null }] },
    {
      id: 'health',
      entries: [
        { id: 'problems', badge: count(counts.problems, 'warning') },
        ...(caps.order ? [{ id: 'load-order' as const, badge: null }] : []),
        ...(caps.startup ? [{ id: 'performance' as const, badge: null }] : []),
      ],
    },
    { id: 'logs', entries: caps.console ? [{ id: 'console', badge: null }] : [] },
  ]
  return groups.filter((g) => g.entries.length > 0)
}

// A section saved from another game or loader that this profile's loader lacks.
function tabUnavailable(tab: TabId, caps: SidebarCaps): boolean {
  return (
    (tab === 'load-order' && !caps.order) ||
    (tab === 'console' && !caps.console) ||
    (tab === 'performance' && !caps.startup)
  )
}

export { type SidebarEntry, type SidebarGroupId, sidebarGroups, tabUnavailable }
