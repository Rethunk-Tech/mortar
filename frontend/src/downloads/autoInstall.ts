import { msg } from '@lingui/core/macro'
import { Events } from '@wailsio/runtime'
import { ExtraFolderMods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useInstall } from '../install/store.ts'
import { gameBusy } from '../launch/busy.ts'
import { useLaunch } from '../launch/store.ts'
import { sameId } from '../mods/lookup.ts'
import { useMods } from '../mods/store.ts'
import { willImport } from '../profiles/gameModsFormat.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

// Installs that a change asked for while the game ran, run once it stops, by the folder that changed.
const waiting = new Map<string, () => void>()
let stopWaiting: (() => void) | null = null

const busy = (game: string) => {
  const { starting, status } = useLaunch.getState()
  return starting || gameBusy(status, game)
}

function whenIdle(game: string, dir: string, run: () => void) {
  if (useProfiles.getState().game?.id !== game) {
    return
  }
  if (!busy(game)) {
    run()
    return
  }
  if (!waiting.has(dir)) {
    useToasts.getState().push({
      kind: 'info',
      title: i18n._(msg`Your build changed; it is added when the game stops`),
      body: dir,
    })
  }
  waiting.set(dir, run)
  stopWaiting ??= useLaunch.subscribe(() => {
    if (busy(game)) {
      return
    }
    stopWaiting?.()
    stopWaiting = null
    const runs = [...waiting.entries()]
    waiting.clear()
    for (const [d, r] of runs) {
      whenIdle(game, d, r)
    }
  })
}

// A mod from the extra mods folder that the open profile already holds is installed again from its folder, so a rebuild
// is what the game loads next; mods only listed there wait for the player to add them.
async function refreshExtraFolderMods(game: string) {
  const listed = (await ExtraFolderMods(game)).mods ?? []
  const held = useMods.getState().mods
  const folders = listed
    .filter(
      (m) => willImport(m.status) && m.folder && m.id && held.some((h) => sameId(h.id, m.id ?? '')),
    )
    .map((m) => m.folder ?? '')
  if (folders.length > 0) {
    await useInstall.getState().installFromExtraFolder(folders)
  }
}

let listening = false

/** Keeps the open profile current with a mod author's builds: a folder set to auto-install is installed each time its
 * contents change, and mods added from the extra mods folder are installed again when theirs does. */
export function initAutoInstall() {
  if (listening || typeof Events.On !== 'function') {
    return
  }
  listening = true
  Events.On('library:auto-install', (event) => {
    const { game, dir } = event.data
    whenIdle(game, dir, () => {
      useInstall.getState().installFromBuildFolder(dir).catch(reportUnexpected)
    })
  })
  Events.On('library:extra-folder', (event) => {
    const game = event.data
    whenIdle(game, `extra:${game}`, () => {
      refreshExtraFolderMods(game).catch(reportUnexpected)
    })
  })
}
