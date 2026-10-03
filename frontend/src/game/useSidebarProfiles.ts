import { useEffect } from 'react'
import { runBackgroundBadgeChecks } from '../mods/badgeDisplay.ts'
import { useBadges } from '../mods/badges.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { orderProfiles } from './profileOrder.ts'

export function useOrderedProfiles(game: string) {
  const allProfiles = useProfiles((s) => s.profiles)
  const profileOrder = useSettings((s) => s.profileOrder)
  const lastPlayedId = useSettings((s) => s.lastPlayed?.[game]?.profile ?? '')
  return {
    allProfiles,
    profiles: orderProfiles(
      allProfiles.filter((p) => !p.hidden),
      profileOrder,
      lastPlayedId,
    ),
  }
}

export function useSidebarBadges(game: string) {
  const { allProfiles, profiles } = useOrderedProfiles(game)
  const loadBadges = useBadges((s) => s.loadAll)
  const stamp = profiles.map((p) => `${p.id}:${String(p.updated)}`).join(',')
  const openId = useProfiles((s) => s.openId)
  const modsShown = useMods((s) => s.loaded && s.modsFor === openId)
  const sidebarBadges = useSettings((s) => s.sidebarBadges)
  const backgroundBadgeChecks = useSettings((s) => s.backgroundBadgeChecks)
  useEffect(() => {
    if (stamp && modsShown && runBackgroundBadgeChecks(sidebarBadges, backgroundBadgeChecks)) {
      loadBadges(game, allProfiles, openId).catch(reportUnexpected)
    }
  }, [
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
