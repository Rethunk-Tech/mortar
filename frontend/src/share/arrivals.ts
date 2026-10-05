import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { ModName } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import { Get } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import type { Arrival } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { Inbox } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { useTab } from '../game/tab.ts'
import { i18n } from '../i18n/index.ts'
import { useDetail } from '../mods/detail.ts'
import { nexusIdOf } from '../mods/lookup.ts'
import { nexusDomain } from '../mods/nexusDomain.ts'
import { nexusModUrl } from '../mods/nexusUrl.ts'
import { useMods } from '../mods/store.ts'
import { isGameId, useNav } from '../nav/store.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { openImport } from './store.ts'

const LINK = 'link'
const MOD = 'mod'

async function openMod(a: Arrival): Promise<void> {
  const game = a.game ?? ''
  const modID = a.modId ?? 0
  if (!isGameId(game) || modID < 1) {
    return
  }
  const settings = await Get()
  useSettings.setState(settings)
  await useProfiles.getState().load(game)
  const last = settings.lastProfile?.[game]
  const profilesState = useProfiles.getState()
  if (last && profilesState.profiles.some((p) => p.id === last && !p.hidden)) {
    profilesState.open(last)
  }
  const profile = openProfileOf(useProfiles.getState())
  useTab.getState().setTab('mods')
  useDetail.getState().show(null)
  let selected = false
  if (profile) {
    await useMods.getState().load()
    const current = openProfileOf(useProfiles.getState())
    const mod = current
      ? useMods.getState().mods.find((candidate) => nexusIdOf(current, candidate) === modID)
      : undefined
    if (mod) {
      useDetail.getState().showAfterLoad(mod)
      selected = true
    }
    const modsState = useMods.getState()
    if (!selected && current && modsState.loaded && modsState.modsFor === current.id) {
      const name = await ModName(game, modID).catch(() => '')
      const label = name || i18n._(msg`Nexus mod ${modID}`)
      useToasts.getState().push({
        kind: 'warning',
        title: i18n._(msg`${label} is not in ${current.name}`),
        action: {
          label: i18n._(msg`Add it`),
          run: () =>
            openImport({
              profileId: current.id,
              link: nexusModUrl(modID, nexusDomain()),
            }),
        },
      })
    }
  }
  useNav.getState().openGame(game)
}

// A link or .mortar file that reached the app from outside only fills the import dialog: the user sees the preview
// and decides. From a game screen, the open profile is offered as the place to add it.
export async function initShare(): Promise<void> {
  const seen = new Set<number>()
  let modRoutes = Promise.resolve()
  const arrive = (a: Arrival) => {
    if (seen.has(a.id)) {
      return
    }
    seen.add(a.id)
    if (a.kind === MOD) {
      modRoutes = modRoutes.then(() => openMod(a)).catch(reportUnexpected)
      return
    }
    const inGame = useNav.getState().route.name === 'game'
    const profileId = inGame ? useProfiles.getState().openId : ''
    openImport(a.kind === LINK ? { profileId, link: a.value } : { profileId, file: a.value })
  }
  Events.On('share:arrived', (e) => arrive(e.data))
  for (const a of (await Inbox()) ?? []) {
    arrive(a)
  }
}
