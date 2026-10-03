import { describe, expect, test } from 'bun:test'
import { useSidebarCollapsed } from '../game/sidebarCollapsed.ts'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useQueue } from '../queue/store.ts'
import { useToasts } from '../toasts/store.ts'
import { runShortcut } from './useShortcuts.ts'

describe('runShortcut', () => {
  test('opens Downloads, notifications, cycles profiles, and toggles the sidebar', () => {
    useQueue.setState({ open: false })
    runShortcut('downloads')
    expect(useQueue.getState().open).toBe(true)

    useToasts.setState({ historyOpen: false })
    runShortcut('notifications')
    expect(useToasts.getState().historyOpen).toBe(true)

    useNav.setState({ route: { name: 'game', game: 'stardew' } })
    useProfiles.setState({
      profiles: [
        { id: 'a', name: 'A', hidden: false },
        { id: 'b', name: 'B', hidden: false },
      ] as never,
      openId: 'a',
      open: (id: string) => useProfiles.setState({ openId: id }),
    })
    runShortcut('next-profile')
    expect(useProfiles.getState().openId).toBe('b')
    runShortcut('previous-profile')
    expect(useProfiles.getState().openId).toBe('a')

    runShortcut('find-all-mods')
    expect(useNav.getState().route.name).toBe('profiles')

    const start = useSidebarCollapsed.getState().collapsed
    runShortcut('collapse-sidebar')
    expect(useSidebarCollapsed.getState().collapsed).toBe(!start)
  })
})
