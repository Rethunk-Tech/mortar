import { useEffect } from 'react'
import { runBackgroundBadgeChecks } from '../mods/badgeDisplay.ts'
import { useBadges } from '../mods/badges.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { orderProfiles } from './profileOrder.ts'

// The profile list still holds the previous game's profiles while a newly opened game loads, so badges wait for it
// rather than ask about one game's profiles under the other's id.
const useListedGame = () => useProfiles((s) => s.game?.id ?? '')

export function useOrderedProfiles(game: string) {
  const allProfiles = useProfiles((s) => s.profiles)
  const profileOrder = useSettings((s) => s.profileOrder)
  const lastPlayedId = useSettings((s) => s.lastPlayed?.[game]?.profile ?? '')
  return {
    allProfiles,
    profiles: orderProfiles(allProfiles, profileOrder, lastPlayedId),
  }
}

export function useSidebarBadges(game: string) {
  const { allProfiles, profiles } = useOrderedProfiles(game)
  const listed = useListedGame() === game
  const loadBadges = useBadges((s) => s.loadAll)
  const stamp = profiles.map((p) => `${p.id}:${String(p.updated)}`).join(',')
  const openId = useProfiles((s) => s.openId)
  const modsShown = useMods((s) => s.loaded && s.modsFor === openId)
  const sidebarBadges = useSettings((s) => s.sidebarBadges)
  const backgroundBadgeChecks = useSettings((s) => s.backgroundBadgeChecks)
  useEffect(() => {
    if (
      listed &&
      stamp &&
      modsShown &&
      runBackgroundBadgeChecks(sidebarBadges, backgroundBadgeChecks)
    ) {
      loadBadges(game, allProfiles, openId).catch(reportUnexpected)
    }
  }, [
    listed,
    game,
    stamp,
    allProfiles,
    loadBadges,
    modsShown,
    openId,
    sidebarBadges,
    backgroundBadgeChecks,
  ])
}

export function useProfilePageBadges(game: string) {
  const allProfiles = useProfiles((s) => s.profiles)
  const listed = useListedGame() === game
  const loadBadges = useBadges((s) => s.loadAll)
  const stamp = allProfiles.map((p) => `${p.id}:${String(p.updated)}`).join(',')
  const sidebarBadges = useSettings((s) => s.sidebarBadges)
  const backgroundBadgeChecks = useSettings((s) => s.backgroundBadgeChecks)
  useEffect(() => {
    if (listed && stamp && runBackgroundBadgeChecks(sidebarBadges, backgroundBadgeChecks)) {
      loadBadges(game, allProfiles, '').catch(reportUnexpected)
    }
  }, [listed, game, stamp, allProfiles, loadBadges, sidebarBadges, backgroundBadgeChecks])
}
