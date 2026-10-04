import { useTab } from '../game/tab.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDetail } from './detail.ts'
import { nexusIdOf } from './lookup.ts'
import { useMods } from './store.ts'

// revealMod opens a profile's Mods tab with one mod selected in the sidebar, matched by its Nexus ID or else its name.
async function revealMod(game: string, profileId: string, nexusId: number, name: string) {
  if (!isGameId(game)) {
    return
  }
  useNav.getState().openGame(game)
  const profiles = useProfiles.getState()
  if (profiles.game?.id !== game) {
    await profiles.load(game)
  }
  useProfiles.getState().open(profileId)
  useTab.getState().setTab('mods')
  await useMods.getState().load()
  const profile = useProfiles.getState().profiles.find((p) => p.id === profileId)
  const { mods } = useMods.getState()
  const mod =
    (profile && nexusId > 0 ? mods.find((m) => nexusIdOf(profile, m) === nexusId) : undefined) ??
    mods.find((m) => m.name === name)
  if (mod) {
    useDetail.getState().show(mod)
  }
}

export const showInProfile = (game: string, profileId: string, nexusId: number, name: string) => {
  revealMod(game, profileId, nexusId, name).catch(reportUnexpected)
}
